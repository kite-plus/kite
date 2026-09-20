// Package frontmatter reads and writes YAML front matter while preserving
// everything it was not asked to change.
//
// A naive unmarshal/marshal round trip reorders keys, drops comments and
// rewrites "tags: [a, b]" as a block list, which turns a one word edit into a
// large diff. That is the most common complaint levelled at git-backed content
// systems, so this package edits a document surgically: an untouched document
// round trips byte for byte, and changing one key rewrites only that key's
// lines.
package frontmatter

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	openDelim  = "---"
	closeAlt   = "..."
	defaultEOL = "\n"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type editKind int

const (
	editReplace editKind = iota
	editInsert
	editDelete
)

type pendingEdit struct {
	kind  editKind
	key   string
	value *yaml.Node
}

// Document is a parsed markdown file: a YAML front matter block plus a body.
type Document struct {
	raw []byte
	bom []byte
	eol string

	hasFM   bool
	fmLines []string
	body    []byte

	root *yaml.Node // mapping node, nil when there is no front matter

	edits       []pendingEdit
	bodyChanged bool
	newBody     []byte
}

// Parse reads a document. A file without front matter is valid: it parses with
// an empty mapping and the whole file as the body.
func Parse(data []byte) (*Document, error) {
	d := &Document{raw: bytes.Clone(data), eol: defaultEOL}

	rest := data
	if after, ok := bytes.CutPrefix(rest, utf8BOM); ok {
		d.bom = utf8BOM
		rest = after
	}
	if i := bytes.IndexByte(rest, '\n'); i > 0 && rest[i-1] == '\r' {
		d.eol = "\r\n"
	}

	lines, terminated := splitLines(rest)
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != openDelim {
		d.body = bytes.Clone(rest)
		d.root = newMapping()
		return d, nil
	}

	closeAt := slices.IndexFunc(lines[1:], func(l string) bool {
		t := strings.TrimRight(l, "\r")
		return t == openDelim || t == closeAlt
	})
	if closeAt >= 0 {
		closeAt++ // compensate for the lines[1:] offset
	}
	if closeAt < 0 {
		return nil, fmt.Errorf("frontmatter: opening %q has no closing delimiter", openDelim)
	}

	d.hasFM = true
	d.fmLines = make([]string, 0, closeAt-1)
	for _, l := range lines[1:closeAt] {
		d.fmLines = append(d.fmLines, strings.TrimRight(l, "\r"))
	}
	d.body = joinLines(lines[closeAt+1:], d.eol, terminated)

	root, err := parseMapping(strings.Join(d.fmLines, "\n"))
	if err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}
	d.root = root
	return d, nil
}

func parseMapping(src string) (*yaml.Node, error) {
	if strings.TrimSpace(src) == "" {
		return newMapping(), nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return newMapping(), nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("front matter must be a mapping, got %s", kindName(root.Kind))
	}
	return root, nil
}

func newMapping() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

// Raw returns the bytes the document was parsed from.
func (d *Document) Raw() []byte { return bytes.Clone(d.raw) }

// HasFrontMatter reports whether the source had a front matter block.
func (d *Document) HasFrontMatter() bool { return d.hasFM }

// Body returns the content after the front matter.
func (d *Document) Body() string {
	if d.bodyChanged {
		return string(d.newBody)
	}
	return string(d.body)
}

// SetBody replaces the content after the front matter.
func (d *Document) SetBody(s string) {
	b := []byte(s)
	if !d.bodyChanged && bytes.Equal(b, d.body) {
		return
	}
	d.bodyChanged = true
	d.newBody = b
}

// Keys returns the top level keys in document order.
func (d *Document) Keys() []string {
	pairs := d.pairs()
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.key)
	}
	return out
}

// Get decodes the value of a top level key.
func (d *Document) Get(key string) (any, bool) {
	for _, p := range d.pairs() {
		if p.key != key {
			continue
		}
		var v any
		if err := p.value.Decode(&v); err != nil {
			return nil, false
		}
		return v, true
	}
	return nil, false
}

// String returns a string valued key, or "" when absent or not a scalar.
func (d *Document) String(key string) string {
	v, ok := d.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// StringSlice returns a key whose value is a list of scalars.
func (d *Document) StringSlice(key string) []string {
	v, ok := d.Get(key)
	if !ok {
		return nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, fmt.Sprint(it))
	}
	return out
}

// Bool returns a boolean valued key.
func (d *Document) Bool(key string) bool {
	v, ok := d.Get(key)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

// Set schedules a key to be written. When the encoded value is equal to what
// is already stored the call is a no-op, so that saving an unchanged document
// produces no diff at all.
func (d *Document) Set(key string, value any) error {
	node := new(yaml.Node)
	if err := node.Encode(value); err != nil {
		return fmt.Errorf("frontmatter: encode %q: %w", key, err)
	}

	if old := d.effectiveValue(key); old != nil {
		same, err := nodesEqual(old, node)
		if err != nil {
			return err
		}
		if same {
			return nil
		}
		adoptStyle(old, node)
		d.edits = append(d.edits, pendingEdit{kind: editReplace, key: key, value: node})
		return nil
	}
	d.edits = append(d.edits, pendingEdit{kind: editInsert, key: key, value: node})
	return nil
}

// SetAll applies Set for each entry, in the given order.
func (d *Document) SetAll(keys []string, values map[string]any) error {
	for _, k := range keys {
		v, ok := values[k]
		if !ok {
			continue
		}
		if err := d.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

// Delete schedules removal of a key. Deleting an absent key is a no-op.
func (d *Document) Delete(key string) {
	if d.effectiveValue(key) == nil {
		return
	}
	d.edits = append(d.edits, pendingEdit{kind: editDelete, key: key})
}

// Dirty reports whether encoding would produce different bytes.
func (d *Document) Dirty() bool { return len(d.edits) > 0 || d.bodyChanged }

// Bytes renders the document.
//
// With no pending edits it returns the original bytes unchanged, which is the
// property that makes "open and save" a no-op in git.
func (d *Document) Bytes() ([]byte, error) {
	if !d.Dirty() {
		return bytes.Clone(d.raw), nil
	}

	lines, err := d.applyEdits()
	if err != nil {
		return nil, err
	}

	body := d.body
	if d.bodyChanged {
		body = d.newBody
	}

	var buf bytes.Buffer
	buf.Write(d.bom)
	if len(lines) > 0 || d.hasFM {
		buf.WriteString(openDelim)
		buf.WriteString(d.eol)
		for _, l := range lines {
			buf.WriteString(l)
			buf.WriteString(d.eol)
		}
		buf.WriteString(openDelim)
		buf.WriteString(d.eol)
	}
	buf.Write(body)
	return buf.Bytes(), nil
}

type pair struct {
	key   string
	kNode *yaml.Node
	value *yaml.Node
}

func (d *Document) pairs() []pair {
	if d.root == nil {
		return nil
	}
	out := make([]pair, 0, len(d.root.Content)/2)
	for i := 0; i+1 < len(d.root.Content); i += 2 {
		out = append(out, pair{
			key:   d.root.Content[i].Value,
			kNode: d.root.Content[i],
			value: d.root.Content[i+1],
		})
	}
	return out
}

// effectiveValue returns the value a key would have after the edits recorded so
// far, or nil when the key would not exist.
func (d *Document) effectiveValue(key string) *yaml.Node {
	var current *yaml.Node
	for _, p := range d.pairs() {
		if p.key == key {
			current = p.value
			break
		}
	}
	for _, e := range d.edits {
		if e.key != key {
			continue
		}
		switch e.kind {
		case editDelete:
			current = nil
		case editReplace, editInsert:
			current = e.value
		}
	}
	return current
}

// span is the inclusive line range a key occupies inside the front matter.
type span struct{ start, end int }

// keySpans maps each top level key to the lines it owns. Trailing blank lines
// and column-zero comments are excluded: those belong to the following key as
// its head comment, and rewriting one key must not disturb its neighbour's.
func (d *Document) keySpans() map[string]span {
	pairs := d.pairs()
	spans := make(map[string]span, len(pairs))
	for i, p := range pairs {
		start := p.kNode.Line - 1
		if start < 0 || start >= len(d.fmLines) {
			continue
		}
		limit := len(d.fmLines) - 1
		if i+1 < len(pairs) {
			limit = pairs[i+1].kNode.Line - 2
		}
		end := start
		for j := min(limit, len(d.fmLines)-1); j >= start; j-- {
			if isDetachedComment(d.fmLines[j]) {
				continue
			}
			end = j
			break
		}
		spans[p.key] = span{start: start, end: end}
	}
	return spans
}

// isDetachedComment reports whether a line is blank or a comment at column
// zero. An indented comment inside a block scalar is content, not a comment.
func isDetachedComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t")
}

func (d *Document) applyEdits() ([]string, error) {
	lines := slices.Clone(d.fmLines)
	spans := d.keySpans()

	// Collapse repeated edits to the same key so each key is rewritten once.
	final := make(map[string]pendingEdit)
	var order []string
	for _, e := range d.edits {
		if _, seen := final[e.key]; !seen {
			order = append(order, e.key)
		}
		final[e.key] = e
	}

	type placed struct {
		span span
		text []string
		del  bool
	}
	var inPlace []placed
	var appended []string

	for _, key := range order {
		e := final[key]
		sp, exists := spans[key]

		switch {
		case e.kind == editDelete:
			if exists {
				inPlace = append(inPlace, placed{span: sp, del: true})
			}
		case exists:
			text, err := encodePair(d.keyNode(key), e.value)
			if err != nil {
				return nil, err
			}
			inPlace = append(inPlace, placed{span: sp, text: text})
		default:
			text, err := encodePair(scalarKey(key), e.value)
			if err != nil {
				return nil, err
			}
			appended = append(appended, text...)
		}
	}

	// Rewrite from the bottom up so earlier spans stay valid.
	slices.SortFunc(inPlace, func(a, b placed) int {
		return cmp.Compare(b.span.start, a.span.start)
	})
	for _, p := range inPlace {
		if p.span.start < 0 || p.span.end >= len(lines) || p.span.start > p.span.end {
			continue
		}
		tail := slices.Clone(lines[p.span.end+1:])
		lines = append(lines[:p.span.start], p.text...)
		lines = append(lines, tail...)
	}

	return append(lines, appended...), nil
}

func (d *Document) keyNode(key string) *yaml.Node {
	for _, p := range d.pairs() {
		if p.key == key {
			return p.kNode
		}
	}
	return scalarKey(key)
}

func scalarKey(key string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
}

// encodePair renders one "key: value" entry. Head and foot comments are
// stripped because they live on lines outside the key's own span and must not
// be duplicated; the line comment stays because it shares the key's line.
func encodePair(k, v *yaml.Node) ([]string, error) {
	kc := *k
	kc.HeadComment, kc.FootComment = "", ""
	vc := *v
	vc.HeadComment, vc.FootComment = "", ""

	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{&kc, &vc}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(m); err != nil {
		return nil, fmt.Errorf("frontmatter: encode %q: %w", k.Value, err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("frontmatter: encode %q: %w", k.Value, err)
	}
	return strings.Split(strings.TrimRight(buf.String(), "\n"), "\n"), nil
}

// adoptStyle carries the old value's formatting over to the replacement, so a
// flow list stays a flow list and a quoted scalar stays quoted.
func adoptStyle(old, replacement *yaml.Node) {
	if old.Kind != replacement.Kind {
		return
	}
	replacement.Style = old.Style
	if old.Kind == yaml.SequenceNode {
		for _, child := range replacement.Content {
			if child.Kind == yaml.ScalarNode && len(old.Content) > 0 {
				child.Style = old.Content[0].Style
			}
		}
	}
}

// nodesEqual compares two nodes by decoded value, ignoring formatting, so that
// rewriting a key with an identical value is recognised as a no-op.
func nodesEqual(a, b *yaml.Node) (bool, error) {
	av, err := canonical(a)
	if err != nil {
		return false, err
	}
	bv, err := canonical(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(av, bv), nil
}

func canonical(n *yaml.Node) ([]byte, error) {
	var v any
	if err := n.Decode(&v); err != nil {
		return nil, fmt.Errorf("frontmatter: decode value: %w", err)
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("frontmatter: canonicalize value: %w", err)
	}
	return out, nil
}

func splitLines(b []byte) (lines []string, terminated bool) {
	if len(b) == 0 {
		return nil, false
	}
	s := string(b)
	terminated = strings.HasSuffix(s, "\n")
	if terminated {
		s = s[:len(s)-1]
	}
	return strings.Split(s, "\n"), terminated
}

func joinLines(lines []string, eol string, terminated bool) []byte {
	if len(lines) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for i, l := range lines {
		buf.WriteString(strings.TrimRight(l, "\r"))
		if i < len(lines)-1 || terminated {
			buf.WriteString(eol)
		}
	}
	return buf.Bytes()
}

func kindName(k yaml.Kind) string {
	switch k {
	case yaml.SequenceNode:
		return "sequence"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return "unknown"
	}
}
