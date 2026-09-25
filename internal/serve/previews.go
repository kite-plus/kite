package serve

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/site"
)

// A preview is kept while it is used. One left alone this long is forgotten,
// and past this many the one used longest ago goes first.
const (
	previewIdle  = 30 * time.Minute
	previewLimit = 8
)

// previews keeps the sites the admin draws with a theme or settings being
// tried, each at an address of its own.
//
// A preview is a whole site rather than one page, so that its pages can be
// followed from one to the next, and its links, stylesheets and pictures are
// the theme's own rather than those of the theme in use.
type previews struct {
	server *Server

	mu   sync.Mutex
	open map[string]*preview
}

// preview is one site being tried, planned the way a build plans the site.
type preview struct {
	base    string
	site    *site.Site
	builder *build.Builder
	routes  *router
	used    time.Time
}

func newPreviews(s *Server) *previews {
	return &previews{server: s, open: make(map[string]*preview)}
}

// Open draws a trial at prefix followed by a new token.
func (p *previews) Open(ctx context.Context, trial api.Trial, prefix string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw[:])
	drawn, err := p.draw(ctx, prefix+token, trial)
	if err != nil {
		return "", err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.forget()
	p.open[token] = drawn
	return token, nil
}

// Update draws an open preview again, at the address it already has.
func (p *previews) Update(ctx context.Context, token string, trial api.Trial) (bool, error) {
	p.mu.Lock()
	current, ok := p.open[token]
	p.mu.Unlock()
	if !ok {
		return false, nil
	}

	drawn, err := p.draw(ctx, current.base, trial)
	if err != nil {
		return true, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// One closed while it was being drawn stays closed.
	if _, still := p.open[token]; still {
		p.open[token] = drawn
	}
	return true, nil
}

// Close forgets a preview.
func (p *previews) Close(token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.open, token)
}

// Serve answers for a page or a file of an open preview.
func (p *previews) Serve(w http.ResponseWriter, r *http.Request, token, path string) bool {
	p.mu.Lock()
	drawn, ok := p.open[token]
	if ok {
		drawn.used = time.Now()
	}
	p.mu.Unlock()
	if !ok {
		return false
	}

	if target, ok := drawn.routes.lookup(path); ok {
		p.render(w, r, drawn, target, http.StatusOK)
		return true
	}
	if within, ok := drawn.site.Resolver.SitePath(path); ok &&
		serveFile(w, r, within, p.server.root, drawn.routes, staticRoots(p.server.root, drawn.site)) {
		return true
	}
	if target, ok := drawn.routes.notFoundTarget(); ok {
		p.render(w, r, drawn, target, http.StatusNotFound)
		return true
	}
	http.NotFound(w, r)
	return true
}

// draw assembles the site a trial describes, at base, and plans it.
func (p *previews) draw(ctx context.Context, base string, trial api.Trial) (*preview, error) {
	s := p.server
	variant, err := s.project().With(site.Variant{
		Theme:    trial.Theme,
		Settings: trial.Settings,
		BaseURL:  base,
	})
	if err != nil {
		return nil, err
	}
	builder, err := variant.Builder(site.BuildOptions{Drafts: s.opts.Drafts, Now: s.now()})
	if err != nil {
		return nil, err
	}
	plan, err := builder.Plan(ctx)
	if err != nil {
		return nil, err
	}
	media, err := build.MediaFiles(plan, os.DirFS(s.root))
	if err != nil {
		return nil, err
	}
	routes := newRouter()
	routes.load(plan)
	routes.loadMedia(media)
	return &preview{base: base, site: variant, builder: builder, routes: routes, used: time.Now()}, nil
}

func (p *previews) render(w http.ResponseWriter, r *http.Request, drawn *preview, t build.Target, status int) {
	html, _, err := drawn.builder.Render(r.Context(), t, render.NewRequest(r))
	if err != nil {
		// A theme being tried may well fail, and saying where is the point
		// of trying it.
		http.Error(w, "render failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", contentLength(len(html)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(html)
	}
}

// forget drops the previews left unused, then the ones used longest ago
// until there is room for one more. The caller holds the lock.
func (p *previews) forget() {
	for token, drawn := range p.open {
		if time.Since(drawn.used) > previewIdle {
			delete(p.open, token)
		}
	}
	for len(p.open) >= previewLimit {
		oldest := ""
		for token, drawn := range p.open {
			if oldest == "" || drawn.used.Before(p.open[oldest].used) {
				oldest = token
			}
		}
		delete(p.open, oldest)
	}
}
