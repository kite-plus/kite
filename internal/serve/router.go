package serve

import (
	"path"
	"strings"
	"sync"

	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render"
)

// router maps a request path to the output a build would have written for it.
//
// Both runtimes therefore render the same target through the same code, which
// makes "the preview matches the deployed site" a property of the design
// rather than something to keep checking by hand.
type router struct {
	mu       sync.RWMutex
	byURL    map[string]build.Target
	byItem   map[content.ID]build.Target
	notFound *build.Target

	// exact holds the pages at old addresses by the spellings a static host
	// serves them at. /about.html is not /about/ to a host, so an old
	// /about.html answers with its own page while /about/ is the item.
	exact map[string]build.Target

	// media maps an output path to the file inside a page bundle it is read
	// from, using the same table the build publishes from.
	media map[string]string
}

func newRouter() *router {
	return &router{byURL: make(map[string]build.Target)}
}

// loadMedia records what the page bundles contribute.
func (r *router) loadMedia(m map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.media = m
}

// bundleFile resolves a request to a file a page bundle owns.
func (r *router) bundleFile(p string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src, ok := r.media[strings.TrimPrefix(path.Clean("/"+p), "/")]
	return src, ok
}

// load replaces the routing table from a plan.
func (r *router) load(p *build.Plan) {
	byURL := make(map[string]build.Target, p.Len())
	byItem := make(map[content.ID]build.Target)
	exact := make(map[string]build.Target)
	var notFound *build.Target

	for _, t := range p.Targets {
		switch t.Kind {
		case render.KindNotFound:
			target := t
			notFound = &target
			continue
		case render.KindAlias:
			for _, spelling := range spellings(t.URL) {
				exact[spelling] = t
			}
			continue
		}
		byURL[normalize(t.URL)] = t
		if t.Kind == render.KindSingle && t.Item != nil {
			byItem[t.Item.ID] = t
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byURL, r.byItem, r.exact, r.notFound = byURL, byItem, exact, notFound
}

// spellings are the request paths a static host serves a page's file at: a
// folder's index at the folder with or without its slash, a file at its own
// name only.
func spellings(link string) []string {
	folder, ok := strings.CutSuffix(link, "/")
	if !ok {
		return []string{link}
	}
	return []string{link, folder, link + "index.html"}
}

// planned returns the target the plan holds for an item, whatever address a
// draft of it would now be given.
func (r *router) planned(id content.ID) (build.Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.byItem[id]
	return t, ok
}

// lookup resolves a request path.
func (r *router) lookup(path string) (build.Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.exact[path]; ok {
		return t, true
	}
	t, ok := r.byURL[normalize(path)]
	return t, ok
}

func (r *router) notFoundTarget() (build.Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.notFound == nil {
		return build.Target{}, false
	}
	return *r.notFound, true
}

func (r *router) size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byURL) + len(r.exact)
}

// normalize reduces the spellings of one address to a single key, the way a
// build does. A preview that disagreed about which of them exist would send
// authors chasing a difference that is not in their content.
func normalize(p string) string { return build.Address(p) }
