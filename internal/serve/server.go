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
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/build"
	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/render"
	"github.com/kite-plus/kite/internal/setup"
	"github.com/kite-plus/kite/internal/site"
	"github.com/kite-plus/kite/web"
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

	// Admin mounts the read model API.
	//
	// It is off unless asked for. The API reports drafts, paths on the host
	// and files the index refused, none of which a publicly reachable server
	// should hand out; local authoring turns it on deliberately.
	Admin bool

	// Auth is who may use the admin. A nil guard, or one with no account
	// behind it, is an open server -- which is why a server that would put
	// one on a public address comes up in setup instead, answering nothing
	// else until it has an owner.
	Auth *auth.Guard

	// Write lets the API change the project.
	//
	// Separate from Admin, and off unless asked for, because reading a
	// repository over HTTP and letting anyone who can reach the port rewrite
	// it are not the same decision.
	Write bool

	Logger *slog.Logger
}

// Server renders a project over HTTP.
type Server struct {
	opts   Options
	router *router
	hub    *reloadHub
	log    *slog.Logger

	// now is injected so that a test can compare a served page against a built
	// one. The clock is an input to rendering like any other, and the design
	// requires every input to be explicit.
	now func() time.Time

	// root never changes, so it is read without the lock.
	root string

	// setup is the first run this server is in the middle of, and nil for
	// every server that is not one.
	setup *setup.Flow

	// reloading lets one reload run at a time, whoever asked for it.
	reloading sync.Mutex

	mu       sync.RWMutex
	site     *site.Site
	builder  *build.Builder
	problems []string
	// due is when the next item dated later falls due and the plan with it
	// stops being current; zero when nothing is waiting.
	due    time.Time
	extras *extras
	// configHash detects a settings change, which needs more than a reindex.
	configHash string
}

// Setup is the first run this server is waiting on, or nil when it is not
// waiting on one. The command that started the server prints from it, so that
// whoever ran it is told where to finish.
func (s *Server) Setup() *setup.Flow { return s.setup }

// project is the site as it currently stands. Configuration can be reloaded
// while requests are in flight, so it is read rather than captured.
func (s *Server) project() *site.Site {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.site
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
	// A studio with no account on an address other than localhost used to be
	// refused outright. A container has no terminal to run
	// `kite auth set-password` in, so that left whoever deployed it with a
	// restart loop and a log line to read.
	//
	// It comes up in setup instead. Nothing but the first-run endpoints
	// answers, so an unconfigured server hands out no content, no drafts and
	// no settings; the form itself is open, and the first browser to reach it
	// gets the account. Giving the server an account before it is reachable
	// is what closes that window -- see [setup] for why it is left open.
	var flow *setup.Flow
	if opts.Admin && !opts.Auth.Required() && !auth.Loopback(opts.Addr) {
		// Setup writes an account and the site's own description, so a
		// read-only server has no way through it and is still refused.
		if !opts.Write {
			return nil, fmt.Errorf("serve: %s can be reached from outside this machine "+
				"and this project has no account, so the studio would be open to anyone;\n"+
				"       set one with `kite auth set-password`, with KITE_ADMIN_USER "+
				"and KITE_ADMIN_PASSWORD,\n"+
				"       or add --write to be guided through setup in a browser", opts.Addr)
		}
		var err error
		if flow, err = setup.New(s.Project.Root, opts.Auth); err != nil {
			return nil, err
		}
	}

	srv := &Server{
		opts:   opts,
		site:   s,
		root:   s.Project.Root,
		setup:  flow,
		router: newRouter(),
		hub:    newReloadHub(),
		log:    opts.Logger,
		now:    now,
	}
	srv.configHash = srv.readConfigHash()
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
	s.reloading.Lock()
	defer s.reloading.Unlock()
	return s.reload(ctx)
}

func (s *Server) reload(ctx context.Context) error {
	// Configuration first: a changed title or theme has to be in place before
	// anything derived from it is rebuilt.
	if err := s.reconfigureIfChanged(); err != nil {
		return err
	}
	current := s.project()

	// A file with no id, or one that will not parse, must not take the server
	// down: the rest of the site still previews, and the page says what was
	// skipped.
	var problems []string
	if _, err := current.Index.Reconcile(ctx); err != nil {
		problems = strings.Split(err.Error(), "\n")
		for _, p := range problems {
			s.log.Warn("not indexed", "detail", p)
		}
	}

	builder, err := current.Builder(site.BuildOptions{
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
	due, err := builder.NextDue(ctx)
	if err != nil {
		return err
	}
	media, err := build.MediaFiles(plan, os.DirFS(s.root))
	if err != nil {
		return err
	}
	s.router.load(plan)
	s.router.loadMedia(media)

	s.mu.Lock()
	s.builder, s.problems, s.due = builder, problems, due
	s.extras = &extras{builder: builder, plan: plan, log: s.log}
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
	if s.opts.Admin {
		api.New(api.Options{
			Site:   s.view,
			Logger: s.log,
			Auth:   s.opts.Auth,
			Setup:  s.setup,
		}).Mount(mux)
		mux.Handle(web.Path+"/", web.Handler())
		mux.Handle(web.Path, http.RedirectHandler(web.Path+"/", http.StatusMovedPermanently))
	}
	mux.HandleFunc("/", s.handle)
	return mux
}

// view is the project as the API should currently see it.
//
// It is read per request rather than captured once, because a reconcile
// replaces the problem list underneath a running server: an admin that held
// the startup copy would keep reporting a file the author has since fixed.
func (s *Server) view() api.View {
	s.mu.RLock()
	problems, current, configHash := s.problems, s.site, s.configHash
	s.mu.RUnlock()

	v := api.View{
		Reader:         current.Reader,
		Resolver:       current.Resolver,
		Types:          current.Project.Types,
		Site:           current.Config.Site,
		Store:          current.Config.Content.Store,
		Runtime:        "serve",
		Theme:          current.Config.Theme.Name,
		Version:        buildinfo.Version,
		ConfigRevision: content.Revision("sha256:" + configHash),
		Problems:       problems,

		ThemeSchema: current.Theme.Manifest.Settings,
		ThemeValues: current.ThemeSettings(),
	}
	if s.opts.Write {
		v.Writer = current.Project.Writer()
		v.Refresh = s.refresh
		v.Publisher = current.Publisher()
	}
	v.Preview = s.preview
	return v
}

// preview renders an item that is not on disk.
//
// It goes through the builder rather than a renderer of its own, so an author
// is looking at the page the build would produce, not at an approximation of
// it that happens to live in the admin.
func (s *Server) preview(ctx context.Context, item *content.Content) ([]byte, error) {
	s.mu.RLock()
	builder, current := s.builder, s.site
	s.mu.RUnlock()

	target := build.Target{
		Kind: render.KindSingle,
		Type: string(item.Kind),
		URL:  current.Resolver.For(item),
		Item: item,
	}
	// A saved item keeps the neighbors the plan gave it. They come from where
	// it sits among the others, which editing its text does not move; a draft
	// that changes its date sees the new ones once it is saved.
	if planned, ok := s.router.planned(item.ID); ok {
		target.Prev, target.Next = planned.Prev, planned.Next
	}
	html, _, err := builder.Render(ctx, target, nil)
	return html, err
}

// reconfigureIfChanged rebuilds the configuration-derived half of the site
// when kite.yaml has changed.
//
// The index is kept: nothing in it depends on configuration, and replacing it
// would pull the database out from under requests already in flight.
func (s *Server) reconfigureIfChanged() error {
	hash := s.readConfigHash()

	s.mu.RLock()
	unchanged := hash == s.configHash
	current := s.site
	s.mu.RUnlock()
	if unchanged {
		return nil
	}

	next, err := current.Reconfigure()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.site, s.configHash = next, hash
	s.mu.Unlock()

	s.log.Info("configuration reloaded")
	return nil
}

func (s *Server) readConfigHash() string {
	data, err := os.ReadFile(filepath.Join(s.root, config.Name))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// refresh reindexes, replans and tells open pages to reload.
//
// It runs before a write is answered rather than being left to the file
// watcher, so that reading back what was just written returns what is on disk
// rather than what the client sent.
func (s *Server) refresh(ctx context.Context) error {
	if err := s.Reload(ctx); err != nil {
		return err
	}
	s.hub.broadcast()
	return nil
}

// catchUp plans again once an item dated later has fallen due.
//
// A plan is made against a frozen clock, and nothing on disk changes when the
// moment a post was waiting for arrives. A server left running would go on
// hiding it until some unrelated edit, so the first request after that moment
// replans before it is answered.
func (s *Server) catchUp(ctx context.Context) {
	if !s.overdue() {
		return
	}
	s.reloading.Lock()
	defer s.reloading.Unlock()
	if !s.overdue() {
		return // a request that got here first has already replanned
	}
	// Every request queued behind this one needs the result, so the client
	// that asked first going away must not abandon it.
	if err := s.reload(context.WithoutCancel(ctx)); err != nil {
		s.log.Error("publishing scheduled content failed", "err", err)
		// Retrying on every request would replan once per page view until
		// someone fixed what broke.
		s.mu.Lock()
		s.due = s.now().Add(time.Minute)
		s.mu.Unlock()
		return
	}
	s.hub.broadcast()
}

func (s *Server) overdue() bool {
	s.mu.RLock()
	due := s.due
	s.mu.RUnlock()
	return !due.IsZero() && !s.now().Before(due)
}

// ListenAndServe runs until the context is canceled.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.opts.Addr)
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	if s.opts.Watch {
		w, err := newWatcher(s.root, s.log)
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
	s.catchUp(r.Context())
	if target, ok := s.router.lookup(r.URL.Path); ok {
		s.renderTarget(w, r, target, http.StatusOK)
		return
	}
	if s.serveExtra(w, r) {
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

	// A page bundle's own files are looked up first, because they are the
	// ones whose address depends on where the page went.
	if src, ok := s.router.bundleFile(rel); ok {
		if data, modTime, found := readFile(os.DirFS(s.root), src); found {
			if ctype := mime.TypeByExtension(path.Ext(rel)); ctype != "" {
				w.Header().Set("Content-Type", ctype)
			}
			w.Header().Set("Cache-Control", "no-store")
			http.ServeContent(w, r, rel, modTime, bytes.NewReader(data))
			return true
		}
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
	current := s.project()

	roots := []fs.FS{os.DirFS(filepath.Join(s.root, "static"))}
	if current.Theme.Static != nil {
		roots = append(roots, current.Theme.Static)
	}
	if current.Theme.Assets != nil {
		roots = append(roots, prefixed{fsys: current.Theme.Assets, prefix: "assets/"})
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
