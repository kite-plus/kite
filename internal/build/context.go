// Package build turns a site into files.
//
// v1 rebuilds everything every time. What v1 does do is record, for each
// output, exactly which inputs it read and which fields of those inputs it
// looked at. Adding the skip check later is then a local change; adding the
// recording later would mean rewriting the render path, which is why it is
// here from the start even though nothing consults it yet.
package build

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// NodeKind classifies a dependency node.
type NodeKind string

const (
	NodeContent  NodeKind = "content"
	NodeTemplate NodeKind = "template"
	NodeAsset    NodeKind = "asset"
	NodeConfig   NodeKind = "config"
	NodeTaxonomy NodeKind = "taxonomy"
	NodeHook     NodeKind = "hook"
)

// Node identifies one input.
type Node struct {
	Kind NodeKind
	ID   string
}

func (n Node) String() string { return string(n.Kind) + ":" + n.ID }

// Dep is one recorded dependency.
//
// Fields records which parts of the node were actually read. Hashing that
// projection rather than the whole object is what makes incremental builds
// worth having: a listing that reads only titles and dates does not become
// stale when a body changes.
type Dep struct {
	Node   Node
	Fields []string
	Hash   string
}

// Recorder collects the dependencies of one output.
type Recorder struct {
	mu   sync.Mutex
	deps map[Node]map[string]struct{}
	hash map[Node]string
}

// NewRecorder returns an empty recorder.
func NewRecorder() *Recorder {
	return &Recorder{deps: make(map[Node]map[string]struct{}), hash: make(map[Node]string)}
}

// Read records that an output read the given fields of a node.
func (r *Recorder) Read(n Node, hash string, fields ...string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deps[n] == nil {
		r.deps[n] = make(map[string]struct{})
	}
	for _, f := range fields {
		r.deps[n][f] = struct{}{}
	}
	if hash != "" {
		r.hash[n] = hash
	}
}

// Deps returns the recorded dependencies in a stable order.
func (r *Recorder) Deps() []Dep {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Dep, 0, len(r.deps))
	for node, fields := range r.deps {
		out = append(out, Dep{
			Node:   node,
			Fields: slices.Sorted(maps.Keys(fields)),
			Hash:   r.hash[node],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Node.String() < out[j].Node.String() })
	return out
}

// CacheKey hashes the recorded dependencies together with the inputs shared by
// every output.
func (r *Recorder) CacheKey(shared ...string) string {
	h := sha256.New()
	// A hash.Hash never returns an error.
	field := func(s string) { _, _ = fmt.Fprintf(h, "%s\x00", s) }

	field(fmt.Sprintf("abi=%d", buildinfo.BuildABIVersion))
	for _, s := range shared {
		field(s)
	}
	for _, d := range r.Deps() {
		field(d.Node.String())
		field(strings.Join(d.Fields, ","))
		field(d.Hash)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Context is the only way anything in a build reads the outside world.
//
// Nothing below the build engine may call os.ReadFile, consult the environment
// or read the clock. Any such hidden input would make an output's cache key a
// lie, and a wrong cache key shows users a stale page, which is worse than a
// slow build.
type Context struct {
	now      time.Time
	recorder *Recorder
	shared   []string
}

// NewContext returns a build context with a frozen clock.
func NewContext(now time.Time, shared ...string) *Context {
	return &Context{now: now.UTC(), recorder: NewRecorder(), shared: shared}
}

// Now returns the frozen build timestamp. Every call during one build returns
// the same instant.
func (c *Context) Now() time.Time { return c.now }

// ForOutput returns a context that records dependencies separately, for
// rendering one output.
func (c *Context) ForOutput() *Context {
	return &Context{now: c.now, recorder: NewRecorder(), shared: c.shared}
}

// Read records a dependency.
func (c *Context) Read(n Node, hash string, fields ...string) {
	c.recorder.Read(n, hash, fields...)
}

// Deps returns what this context recorded.
func (c *Context) Deps() []Dep { return c.recorder.Deps() }

// CacheKey returns the cache key of the output this context rendered.
func (c *Context) CacheKey() string { return c.recorder.CacheKey(c.shared...) }
