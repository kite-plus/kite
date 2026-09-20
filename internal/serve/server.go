// Package serve renders a site per request instead of building it to disk.
//
// This is the file-backed half of the serve runtime. Nothing here renders
// anything itself: a request is resolved to the output a build would have
// written for that URL, and the build engine produces it. The two runtimes
// share a read model, a renderer and a URL resolver, so a preview cannot
// quietly disagree with the deployed site.
package serve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/site"
)

// Options configures a server.
type Options struct {
	Addr string

	// LiveReload injects a script that reloads open pages after a change.
	//
	// It is the one thing a served page carries that a built one does not, so
	// it is switchable: with it off the response is the built file, byte for
	// byte, which is what the parity test relies on.
	LiveReload bool

	// Watch reindexes and replans when the project changes.
	Watch bool

	// Drafts renders unpublished content, which a preview usually wants and a
	// deployment never does.
	Drafts bool

	Logger *slog.Logger
}

// Server renders a project over HTTP.
type Server struct {
	opts   Options
	site   *site.Site
	router *router
	hub    *reloadHub
	log    *slog.Logger

	// now is injected so that a test can compare a served page against a built
	// one. The clock is an input to rendering like any other, and the design
	// requires every input to be explicit.
	now func() time.Time

	mu       sync.RWMutex
	builder  *build.Builder
	problems []string
}

// New prepares a server over an opened project.
func New(ctx context.Context, s *site.Site, opts Options) (*Server, error) {
	return NewWithClock(ctx, s, opts, time.Now)
}

// NewWithClock is [New] with the clock supplied, for tests that need a
// rendering to be comparable rather than current.
func NewWithClock(ctx context.Context, s *site.Site, opts Options, now func() time.Time) (*Server, error) {
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:1717"
	}
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	srv := &Server{
		opts:   opts,
		site:   s,
		router: newRouter(),
		hub:    newReloadHub(),
		log:    opts.Logger,
		now:    now,
	}
	if err := srv.Reload(ctx); err != nil {
		return nil, err
	}
	return srv, nil
}

// Reload reindexes the project and rebuilds the routing table.
//
// The clock is read once here rather than per request, so every page rendered
// between two changes agrees about what time it is, the way every page in one
// build does.
func (s *Server) Reload(ctx context.Context) error {
	// A file with no id, or one that will not parse, must not take the server
	// down: the rest of the site still previews, and the page says what was
	// skipped.
	var problems []string
	if _, err := s.site.Index.Reconcile(ctx); err != nil {
		problems = strings.Split(err.Error(), "\n")
		for _, p := range problems {
			s.log.Warn("not indexed", "detail", p)
		}
	}

	builder, err := s.site.Builder(site.BuildOptions{
		Drafts: s.opts.Drafts,
		Now:    s.now(),
	})
	if err != nil {
		return err
	}

	plan, err := builder.Plan(ctx)
	if err != nil {
		return err
	}
	s.router.load(plan)

	s.mu.Lock()
	s.builder, s.problems = builder, problems
	s.mu.Unlock()

	s.log.Info("site loaded", "routes", s.router.size())
	return nil
}

// Handler returns the HTTP handler, which is also what the tests drive.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	if s.opts.LiveReload {
		mux.HandleFunc("GET "+ReloadPath, s.hub.handleReload)
	}
	mux.HandleFunc("/", s.handle)
	return mux
}

// ListenAndServe runs until the context is canceled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	if s.opts.Watch {
		w, err := newWatcher(s.site.Project.Root, s.log)
		if err != nil {
			return err
		}
		defer func() { _ = w.Close() }()
		go w.run(ctx, func() {
			if err := s.Reload(ctx); err != nil {
				s.log.Error("reload failed", "err", err)
				return
			}
			s.hub.broadcast()
		})
	}

	httpSrv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdown)
	}()

	s.log.Info("listening", "addr", "http://"+listener.Addr().String())
	if err := httpSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// URL is the address the server listens on.
func (s *Server) URL() string { return "http://" + s.opts.Addr }

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if target, ok := s.router.lookup(r.URL.Path); ok {
		s.renderTarget(w, r, target, http.StatusOK)
		return
	}
	if s.serveStatic(w, r) {
		return
	}
	if target, ok := s.router.notFoundTarget(); ok {
		s.renderTarget(w, r, target, http.StatusNotFound)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) renderTarget(w http.ResponseWriter, r *http.Request, t build.Target, status int) {
	s.mu.RLock()
	builder, problems := s.builder, s.problems
	s.mu.RUnlock()

	html, _, err := builder.Render(r.Context(), t, render.NewRequest(r))
	if err != nil {
		s.log.Error("render failed", "url", r.URL.Path, "err", err)
		http.Error(w, "render failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if s.opts.LiveReload {
		html = inject(html, problems)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", contentLength(len(html)))
	// A preview that served a stale page from the browser cache would defeat
	// the point of watching the filesystem at all.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	if r.Method != http.MethodHead {
		_, _ = w.Write(html)
	}
}

// serveStatic answers for files a build would copy rather than render.
func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) bool {
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}

	for _, root := range s.staticRoots() {
		data, modTime, ok := readFile(root, rel)
		if !ok {
			continue
		}
		if ctype := mime.TypeByExtension(path.Ext(rel)); ctype != "" {
			w.Header().Set("Content-Type", ctype)
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, rel, modTime, bytes.NewReader(data))
		return true
	}
	return false
}

// staticRoots lists where unprocessed files live, in the order a build copies
// them: the project's own static directory wins over the theme's.
func (s *Server) staticRoots() []fs.FS {
	roots := []fs.FS{os.DirFS(filepath.Join(s.site.Project.Root, "static"))}
	if s.site.Theme.Static != nil {
		roots = append(roots, s.site.Theme.Static)
	}
	if s.site.Theme.Assets != nil {
		roots = append(roots, prefixed{fsys: s.site.Theme.Assets, prefix: "assets/"})
	}
	return roots
}

// readFile loads a small file whole, which keeps the handler free of the type
// assertions a seekable fs.File would otherwise need.
func readFile(fsys fs.FS, name string) ([]byte, time.Time, bool) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, time.Time{}, false
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return nil, time.Time{}, false
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, time.Time{}, false
	}
	return data, info.ModTime(), true
}

// prefixed serves a filesystem under a URL prefix, mirroring where a build
// copies the theme's assets.
type prefixed struct {
	fsys   fs.FS
	prefix string
}

func (p prefixed) Open(name string) (fs.File, error) {
	rest, ok := strings.CutPrefix(name, p.prefix)
	if !ok {
		return nil, fs.ErrNotExist
	}
	return p.fsys.Open(rest)
}
