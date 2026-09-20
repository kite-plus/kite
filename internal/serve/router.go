package serve

import (
	"strings"
	"sync"

	"github.com/kite-plus/kite/internal/build"
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
	notFound *build.Target
}

func newRouter() *router {
	return &router{byURL: make(map[string]build.Target)}
}

// load replaces the routing table from a plan.
func (r *router) load(p *build.Plan) {
	byURL := make(map[string]build.Target, p.Len())
	var notFound *build.Target

	for _, t := range p.Targets {
		if t.Kind == render.KindNotFound {
			target := t
			notFound = &target
			continue
		}
		byURL[normalize(t.URL)] = t
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byURL, r.notFound = byURL, notFound
}

// lookup resolves a request path.
func (r *router) lookup(path string) (build.Target, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
	return len(r.byURL)
}

// normalize reduces the spellings of one address to a single key.
//
// A static host serves /posts/hello/, /posts/hello and /posts/hello/index.html
// as the same page. A preview that disagreed about which of those exist would
// send authors chasing a difference that is not in their content.
func normalize(p string) string {
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	p = strings.TrimSuffix(p, "index.html")
	p = "/" + strings.Trim(p, "/")
	if p == "/" {
		return p
	}
	return strings.TrimSuffix(p, ".html")
}
