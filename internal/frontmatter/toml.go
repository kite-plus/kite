package frontmatter

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// tomlDelim opens and closes TOML front matter, as Hugo writes it.
const tomlDelim = "+++"

// tomlFront is TOML front matter, held to the promise the YAML editor keeps:
// a document nobody edited round trips byte for byte, and changing one key
// rewrites only that key's lines.
//
// Values are decoded by a TOML library. Which lines each key owns is worked
// out here, since no decoder reports it. When a document uses TOML this cannot
// place, it can still be read, and only an edit to it fails.
type tomlFront struct {
	lines  []string
	values map[string]any
	keys   []string

	regions map[string][]tomlRegion
	// tables is the line of the first table header, or len(lines): a key
	// written after it would belong to that table.
	tables  int
	scanErr error

	edits []tomlEdit
}

// tomlRegion is lines a top level key occupies.
type tomlRegion struct {
	start, end int

	// table marks a [table] section or a dotted key such as params.cover,
	// which cannot be rewritten as a single "key = value".
	table bool

	// prefix is the line up to the value, "title = " as the author spaced
	// it, and suffix a comment after the value; a rewrite keeps both.
	prefix, suffix string
	style          tomlStyle
}

// tomlStyle is how a value was written, for a replacement to follow.
type tomlStyle int

const (
	tomlPlain tomlStyle = iota
	tomlLiteral
	tomlLocalDate
	tomlLocalDateTime
)

type tomlEdit struct {
	key    string
	value  any
	delete bool
}

func parseTOML(block []string) (*tomlFront, error) {
	f := &tomlFront{lines: block}
	var decoded map[string]any
	if err := toml.Unmarshal([]byte(strings.Join(block, "\n")), &decoded); err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}
	f.values = make(map[string]any, len(decoded))
	for k, v := range decoded {
		f.values[k] = tomlNormal(v)
	}
	f.keys, f.regions, f.tables, f.scanErr = scanTOML(block)
	if f.scanErr != nil {
		f.keys = slices.Sorted(maps.Keys(f.values))
	}
	return f, nil
}

func (f *tomlFront) get(key string) (any, bool) {
	v, ok := f.values[key]
	for _, e := range f.edits {
		if e.key == key {
			v, ok = e.value, !e.delete
		}
	}
	return v, ok
}

func (f *tomlFront) set(key string, value any) {
	if old, ok := f.get(key); ok && tomlEqual(old, value) {
		return
	}
	f.edits = append(f.edits, tomlEdit{key: key, value: value})
}

func (f *tomlFront) delete(key string) {
	if _, ok := f.get(key); ok {
		f.edits = append(f.edits, tomlEdit{key: key, delete: true})
	}
}

// apply renders the block with every edit made.
func (f *tomlFront) apply() ([]string, error) {
	if f.scanErr != nil {
		return nil, fmt.Errorf("frontmatter: cannot edit this TOML: %w", f.scanErr)
	}

	final := make(map[string]tomlEdit)
	var order []string
	for _, e := range f.edits {
		if _, seen := final[e.key]; !seen {
			order = append(order, e.key)
		}
		final[e.key] = e
	}

	type change struct {
		region tomlRegion
		text   []string
	}
	var changes []change
	var inserted, tables []string
	for _, key := range order {
		e := final[key]
		regions := f.regions[key]
		inline := !e.delete && len(regions) == 1 && !regions[0].table
		if inline {
			r := regions[0]
			text, err := encodeTOMLValue(e.value, r.style)
			if err == nil {
				changes = append(changes, change{region: r, text: []string{r.prefix + text + r.suffix}})
				continue
			}
		}
		// Anything else goes away where it was, and a value that is kept
		// is written again where it now belongs.
		for _, r := range regions {
			// A table between two blank lines takes one of them with it,
			// rather than leaving the two to meet.
			if r.table && r.start > 0 && strings.TrimSpace(f.lines[r.start-1]) == "" &&
				(r.end+1 == len(f.lines) || strings.TrimSpace(f.lines[r.end+1]) == "") {
				r.start--
			}
			changes = append(changes, change{region: r})
		}
		if e.delete {
			continue
		}
		if text, err := encodeTOMLValue(e.value, tomlPlain); err == nil {
			inserted = append(inserted, tomlKey(key)+" = "+text)
			continue
		}
		section, err := toml.Marshal(map[string]any{key: e.value})
		if err != nil {
			return nil, fmt.Errorf("frontmatter: encode %q: %w", key, err)
		}
		tables = append(tables, "")
		tables = append(tables, strings.Split(strings.TrimRight(string(section), "\n"), "\n")...)
	}

	lines := slices.Clone(f.lines)
	// New keys go after the last top level key, before any table.
	at := 0
	for _, regions := range f.regions {
		for _, r := range regions {
			if !r.table && r.start < f.tables {
				at = max(at, r.end+1)
			}
		}
	}
	changes = append(changes, change{region: tomlRegion{start: at, end: at - 1}, text: inserted})

	// From the bottom up, so each change leaves the lines above it in place.
	slices.SortStableFunc(changes, func(a, b change) int { return b.region.start - a.region.start })
	for _, c := range changes {
		tail := slices.Clone(lines[c.region.end+1:])
		lines = append(append(lines[:c.region.start], c.text...), tail...)
	}
	if len(tables) > 0 {
		if len(lines) == 0 {
			tables = tables[1:]
		}
		lines = append(lines, tables...)
	}
	return lines, nil
}

// scanTOML finds the lines every top level key owns.
func scanTOML(lines []string) (keys []string, regions map[string][]tomlRegion, tables int, err error) {
	regions = make(map[string][]tomlRegion)
	tables = len(lines)
	add := func(key string, r tomlRegion) {
		if _, ok := regions[key]; !ok {
			keys = append(keys, key)
		}
		regions[key] = append(regions[key], r)
	}

	table, from := "", -1
	closeTable := func(end int) {
		if table == "" {
			return
		}
		// Blank lines and comments at the end belong to whatever follows.
		for end > from && tomlBlank(lines[end]) {
			end--
		}
		add(table, tomlRegion{start: from, end: end, table: true})
	}

	for i := 0; i < len(lines); {
		line := lines[i]
		p := skipSpace(line, 0)
		switch {
		case p == len(line) || line[p] == '#':
			i++
		case line[p] == '[':
			closeTable(i - 1)
			open := p + 1
			if open < len(line) && line[open] == '[' {
				open++
			}
			name, _, _, err := tomlKeyPath(line, open)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("line %d: %w", i+1, err)
			}
			table, from = name, i
			tables = min(tables, i)
			i++
		default:
			first, parts, eq, err := tomlKeyPath(line, p)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("line %d: %w", i+1, err)
			}
			if eq >= len(line) || line[eq] != '=' {
				return nil, nil, 0, fmt.Errorf("line %d: expected = after %q", i+1, first)
			}
			valueAt := skipSpace(line, eq+1)
			end, after, err := tomlValueEnd(lines, i, valueAt)
			if err != nil {
				return nil, nil, 0, fmt.Errorf("line %d: %w", i+1, err)
			}
			if table == "" {
				r := tomlRegion{start: i, end: end, table: parts > 1, prefix: line[:valueAt], style: tomlStyleOf(line[valueAt:])}
				if rest := lines[end][after:]; strings.Contains(rest, "#") {
					r.suffix = rest
				}
				add(first, r)
			}
			i = end + 1
		}
	}
	closeTable(len(lines) - 1)
	return keys, regions, tables, nil
}

// tomlKeyPath reads a key, dotted or not, from s[p:]. It returns the first
// part of it, how many parts it has, and the position just past it.
func tomlKeyPath(s string, p int) (string, int, int, error) {
	var first string
	for part := 0; ; part++ {
		p = skipSpace(s, p)
		var name string
		switch {
		case p < len(s) && s[p] == '"':
			end, err := tomlStringEnd(s, p)
			if err != nil {
				return "", 0, 0, err
			}
			unquoted, err := strconv.Unquote(s[p:end])
			if err != nil {
				return "", 0, 0, fmt.Errorf("a quoted key cannot be read: %w", err)
			}
			name, p = unquoted, end
		case p < len(s) && s[p] == '\'':
			q := strings.IndexByte(s[p+1:], '\'')
			if q < 0 {
				return "", 0, 0, errors.New("a quoted key is not closed")
			}
			name, p = s[p+1:p+1+q], p+2+q
		default:
			start := p
			for p < len(s) && tomlBare(s[p]) {
				p++
			}
			if p == start {
				return "", 0, 0, errors.New("expected a key")
			}
			name = s[start:p]
		}
		if part == 0 {
			first = name
		}
		p = skipSpace(s, p)
		if p < len(s) && s[p] == '.' {
			p++
			continue
		}
		return first, part + 1, p, nil
	}
}

// tomlValueEnd finds where a value starting at column p of line i ends: the
// line it ends on and the column just after it there.
func tomlValueEnd(lines []string, i, p int) (int, int, error) {
	depth := 0
	for line := i; line < len(lines); line, p = line+1, 0 {
		s := lines[line]
		for p < len(s) {
			switch {
			case strings.HasPrefix(s[p:], `"""`), strings.HasPrefix(s[p:], `'''`):
				end, at, err := tomlMultilineEnd(lines, line, p)
				if err != nil {
					return 0, 0, err
				}
				line, p, s = end, at, lines[end]
				continue
			case s[p] == '"':
				end, err := tomlStringEnd(s, p)
				if err != nil {
					return 0, 0, err
				}
				p = end
				continue
			case s[p] == '\'':
				q := strings.IndexByte(s[p+1:], '\'')
				if q < 0 {
					return 0, 0, errors.New("a string is not closed")
				}
				p += q + 2
				continue
			case s[p] == '[' || s[p] == '{':
				depth++
			case s[p] == ']' || s[p] == '}':
				depth--
			case s[p] == '#':
				if depth == 0 {
					return line, strings.LastIndexFunc(s[:p], func(r rune) bool { return r != ' ' && r != '\t' }) + 1, nil
				}
				p = len(s) // a comment inside an array runs to the end of its line
				continue
			}
			p++
		}
		if depth <= 0 {
			return line, len(strings.TrimRight(s, " \t")), nil
		}
	}
	return 0, 0, errors.New("a value is not closed before the end of the front matter")
}

// tomlStringEnd returns the position just past a basic string that starts at
// s[p], which is a double quote.
func tomlStringEnd(s string, p int) (int, error) {
	for q := p + 1; q < len(s); q++ {
		switch s[q] {
		case '\\':
			q++
		case '"':
			return q + 1, nil
		}
	}
	return 0, errors.New("a string is not closed")
}

// tomlMultilineEnd finds the end of a multi-line string that opens at s[p].
// Up to two quotes may sit against the closing delimiter as part of the
// string, so the delimiter is the last three of the run.
func tomlMultilineEnd(lines []string, line, p int) (int, int, error) {
	delim := lines[line][p : p+3]
	escapes := delim == `"""`
	p += 3
	for ; line < len(lines); line, p = line+1, 0 {
		s := lines[line]
		for ; p < len(s); p++ {
			if escapes && s[p] == '\\' {
				p++
				continue
			}
			if strings.HasPrefix(s[p:], delim) {
				run := 0
				for p+run < len(s) && s[p+run] == delim[0] {
					run++
				}
				return line, p + min(run, 5), nil
			}
		}
	}
	return 0, 0, errors.New("a multi-line string is not closed")
}

func tomlStyleOf(value string) tomlStyle {
	v := strings.TrimSpace(value)
	if strings.HasPrefix(v, "'") && !strings.HasPrefix(v, "'''") {
		return tomlLiteral
	}
	if len(v) < 10 || !tomlDigits(v[0:4]) || v[4] != '-' || !tomlDigits(v[5:7]) || v[7] != '-' || !tomlDigits(v[8:10]) {
		return tomlPlain
	}
	// A time follows the date after a T, or after a single space.
	rest := v[10:]
	if len(rest) < 3 || (rest[0] != 'T' && rest[0] != 't' && rest[0] != ' ') || !tomlDigits(rest[1:3]) {
		return tomlLocalDate
	}
	clock := rest[1:]
	if i := strings.IndexAny(clock, " \t#"); i >= 0 {
		clock = clock[:i]
	}
	// An offset is a Z, or a sign after the hours, minutes and seconds.
	if strings.ContainsAny(clock, "Zz+") || strings.Contains(clock[min(len(clock), 8):], "-") {
		return tomlPlain
	}
	return tomlLocalDateTime
}

func tomlDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

// encodeTOMLValue renders a value that fits on one line, following the style
// the old value was written in where the new one allows it.
func encodeTOMLValue(v any, style tomlStyle) (string, error) {
	switch x := v.(type) {
	case string:
		if style == tomlLiteral && !strings.ContainsAny(x, "'\r\n") {
			return "'" + x + "'", nil
		}
		return tomlQuote(x), nil
	case bool:
		return strconv.FormatBool(x), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(x), nil
	case float32:
		return tomlFloat(float64(x)), nil
	case float64:
		return tomlFloat(x), nil
	case time.Time:
		midnight := x.Hour() == 0 && x.Minute() == 0 && x.Second() == 0 && x.Nanosecond() == 0
		switch {
		case style == tomlLocalDate && midnight:
			return x.Format("2006-01-02"), nil
		case style == tomlLocalDateTime:
			return x.Format("2006-01-02T15:04:05"), nil
		}
		return x.Format(time.RFC3339Nano), nil
	case []string:
		out := make([]string, len(x))
		for i, s := range x {
			out[i] = tomlQuote(s)
		}
		return "[" + strings.Join(out, ", ") + "]", nil
	case []any:
		out := make([]string, len(x))
		for i, item := range x {
			text, err := encodeTOMLValue(item, tomlPlain)
			if err != nil {
				return "", err
			}
			out[i] = text
		}
		return "[" + strings.Join(out, ", ") + "]", nil
	}
	return "", fmt.Errorf("frontmatter: %T does not fit on one line", v)
}

func tomlFloat(f float64) string {
	switch {
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	case math.IsNaN(f):
		return "nan"
	}
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEn") {
		s += ".0"
	}
	return s
}

// tomlQuote writes a basic string. Go's own quoting is close but not the same
// language: it would write \x escapes, which TOML does not have.
func tomlQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\f':
			b.WriteString(`\f`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04X`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// tomlKey writes a key bare when it can be, quoted when it cannot.
func tomlKey(key string) string {
	for i := 0; i < len(key); i++ {
		if !tomlBare(key[i]) {
			return tomlQuote(key)
		}
	}
	if key == "" {
		return `""`
	}
	return key
}

func tomlBare(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}

func tomlBlank(line string) bool {
	t := strings.TrimSpace(line)
	return t == "" || strings.HasPrefix(t, "#")
}

func skipSpace(s string, p int) int {
	for p < len(s) && (s[p] == ' ' || s[p] == '\t') {
		p++
	}
	return p
}

// tomlNormal puts decoded and assigned values in one shape, so that a value
// set to what is already there is recognized as no change. Dates without a
// zone are read as UTC, the zone a site builds in unless it says otherwise.
func tomlNormal(v any) any {
	switch x := v.(type) {
	case toml.LocalDate:
		return time.Date(x.Year, time.Month(x.Month), x.Day, 0, 0, 0, 0, time.UTC)
	case toml.LocalDateTime:
		return time.Date(x.Year, time.Month(x.Month), x.Day, x.Hour, x.Minute, x.Second, x.Nanosecond, time.UTC)
	case toml.LocalTime:
		return x.String()
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = tomlNormal(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = tomlNormal(item)
		}
		return out
	}
	return v
}

func tomlEqual(a, b any) bool {
	a, b = tomlComparable(tomlNormal(a)), tomlComparable(tomlNormal(b))
	if at, ok := a.(time.Time); ok {
		bt, ok := b.(time.Time)
		return ok && at.Equal(bt)
	}
	return reflect.DeepEqual(a, b)
}

// tomlComparable widens numbers and lists so that 3 and int64(3), or
// []string and []any of the same strings, compare equal.
func tomlComparable(v any) any {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case float32:
		return float64(x)
	case []string:
		out := make([]any, len(x))
		for i, s := range x {
			out[i] = s
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = tomlComparable(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, item := range x {
			out[k] = tomlComparable(item)
		}
		return out
	}
	return v
}
