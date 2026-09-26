// Package hook is the extension point bus.
//
// Kite's own sitemap, feed and page post-processing are registered here rather
// than called directly from the build engine. Using the bus for built-in work
// from the start is the only reliable way to find out whether the extension
// points are sufficient: a hook set designed in the abstract and first used by
// third parties much later is a hook set that has to be redesigned.
//
// Every hook is batch shaped, taking a whole document or a whole build. A
// per-node interface would be unusable once hooks cross a WebAssembly
// boundary, where each crossing costs an allocation and two copies.
package hook

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/kite-plus/kite/internal/content"
)

// Phase says when a hook may run.
type Phase int

const (
	// PhaseBuild hooks run during a build and must be pure: given the same
	// inputs they must produce the same output, read no clock, and perform no
	// I/O of their own. Impurity here makes incremental builds unsound.
	PhaseBuild Phase = iota

	// PhaseRequest hooks run while serving and may have side effects.
	PhaseRequest
)

func (p Phase) String() string {
	if p == PhaseBuild {
		return "build"
	}
	return "request"
}

// Hook is the metadata every extension carries.
type Hook interface {
	// Name identifies the hook in logs and in error messages.
	Name() string

	Phase() Phase

	// CacheKey contributes to the build cache key of anything this hook
	// touches. Without it, upgrading a hook would leave stale pages in place,
	// which is worse than a slow build.
	CacheKey() []byte
}

// MarkdownDoc is the source of one item, before it is rendered.
type MarkdownDoc struct {
	Item   *content.Content
	Source string
}

// HTMLDoc is one rendered document.
type HTMLDoc struct {
	Item *content.Content
	URL  string
	HTML string

	// Kind is the kind of page: home, single, list, taxonomy, term or
	// notFound, for a hook that belongs on some pages and not others.
	Kind string
}

// PageInfo describes a page that has just been written.
type PageInfo struct {
	Item       *content.Content
	URL        string
	OutputPath string
	Title      string
	Excerpt    string

	// Kind is the kind of page, as in HTMLDoc.
	Kind string

	// Text is the plain text of the item's body, a block to a line, and
	// empty on a page of no item.
	Text string

	// Indexable says whether the page belongs in listings that describe the
	// site to the outside world. An error page is rendered and written but is
	// not part of the site's contents.
	Indexable bool
}

// SiteInfo is the part of the site description a hook may read.
type SiteInfo struct {
	Title       string
	Description string
	BaseURL     string
	Language    string
}

// BuildInfo is the whole build, handed to hooks once every page is rendered.
type BuildInfo struct {
	Site  SiteInfo
	Pages []PageInfo

	// Emit writes an extra output file. It is the only way a hook may produce
	// output, so that everything written during a build stays accounted for.
	Emit func(path string, data []byte) error
}

// MarkdownTransformer rewrites an item's source before it is parsed.
type MarkdownTransformer interface {
	Hook
	TransformMarkdown(ctx context.Context, doc *MarkdownDoc) error
}

// HTMLTransformer rewrites a rendered document.
type HTMLTransformer interface {
	Hook
	TransformHTML(ctx context.Context, doc *HTMLDoc) error
}

// PageObserver is told about each rendered page, for hooks that accumulate
// state such as a search index.
type PageObserver interface {
	Hook
	PageRendered(ctx context.Context, page *PageInfo) error
}

// BuildCompleter runs once, after every page is rendered.
type BuildCompleter interface {
	Hook
	BuildComplete(ctx context.Context, build *BuildInfo) error
}

type registration struct {
	hook     Hook
	priority int
	order    int
}

// Bus holds the registered hooks.
type Bus struct {
	mu      sync.RWMutex
	entries []registration
	next    int
}

// NewBus returns an empty bus.
func NewBus() *Bus { return &Bus{} }

// DefaultPriority is the priority of a hook registered without an opinion.
const DefaultPriority = 100

// Register adds a hook. Lower priorities run first; ties break on registration
// order, so a given set of hooks always runs in the same sequence and the
// build stays reproducible.
func (b *Bus) Register(h Hook, priority int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, registration{hook: h, priority: priority, order: b.next})
	b.next++
	slices.SortStableFunc(b.entries, func(x, y registration) int {
		return cmp.Or(cmp.Compare(x.priority, y.priority), cmp.Compare(x.order, y.order))
	})
}

// Hooks returns every registered hook in execution order.
func (b *Bus) Hooks() []Hook {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Hook, 0, len(b.entries))
	for _, e := range b.entries {
		out = append(out, e.hook)
	}
	return out
}

// CacheKey returns the combined key of every build-phase hook, for inclusion
// in the build cache key.
func (b *Bus) CacheKey() []byte {
	var key []byte
	for _, h := range b.Hooks() {
		if h.Phase() != PhaseBuild {
			continue
		}
		key = append(key, h.Name()...)
		key = append(key, 0)
		key = append(key, h.CacheKey()...)
		key = append(key, 0)
	}
	return key
}

// TransformMarkdown runs every markdown transformer in order.
func (b *Bus) TransformMarkdown(ctx context.Context, doc *MarkdownDoc) error {
	for _, h := range b.Hooks() {
		t, ok := h.(MarkdownTransformer)
		if !ok {
			continue
		}
		if err := t.TransformMarkdown(ctx, doc); err != nil {
			return wrap(h, err)
		}
	}
	return nil
}

// TransformHTML runs every HTML transformer in order.
func (b *Bus) TransformHTML(ctx context.Context, doc *HTMLDoc) error {
	for _, h := range b.Hooks() {
		t, ok := h.(HTMLTransformer)
		if !ok {
			continue
		}
		if err := t.TransformHTML(ctx, doc); err != nil {
			return wrap(h, err)
		}
	}
	return nil
}

// PageRendered notifies every page observer.
func (b *Bus) PageRendered(ctx context.Context, page *PageInfo) error {
	for _, h := range b.Hooks() {
		o, ok := h.(PageObserver)
		if !ok {
			continue
		}
		if err := o.PageRendered(ctx, page); err != nil {
			return wrap(h, err)
		}
	}
	return nil
}

// BuildComplete runs every completion hook.
func (b *Bus) BuildComplete(ctx context.Context, build *BuildInfo) error {
	for _, h := range b.Hooks() {
		c, ok := h.(BuildCompleter)
		if !ok {
			continue
		}
		if err := c.BuildComplete(ctx, build); err != nil {
			return wrap(h, err)
		}
	}
	return nil
}

// Error names the hook that failed, so a broken extension is never mistaken
// for a broken core.
type Error struct {
	Hook string
	Err  error
}

func (e *Error) Error() string { return "hook " + e.Hook + ": " + e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func wrap(h Hook, err error) error { return &Error{Hook: h.Name(), Err: err} }

// Base is a convenience embed for hooks with no cache-relevant configuration.
type Base struct {
	HookName    string
	HookPhase   Phase
	HookVersion string
}

func (b Base) Name() string     { return b.HookName }
func (b Base) Phase() Phase     { return b.HookPhase }
func (b Base) CacheKey() []byte { return []byte(b.HookVersion) }
