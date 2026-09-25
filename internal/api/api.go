// Package api serves the read model over HTTP.
//
// The admin has no private channel into the core: it calls the same endpoints
// a third-party client would. That costs a little — admin-only features have
// to be designed as public API too — and buys the guarantee that anything the
// admin can do, a script or another front end can also do.
package api

import (
	"log/slog"
	"net/http"

	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/setup"
)

// Prefix is where the API is mounted. It is versioned in the path so that a
// breaking change can be introduced alongside, never in place.
const Prefix = "/api/v1"

// Options configures a server.
type Options struct {
	Site   SiteSource
	Logger *slog.Logger

	// Auth decides who may call. A guard with no account behind it is an
	// open server, which is what a local preview of a project that has never
	// had a password is.
	Auth *auth.Guard

	// Setup is the first run this server is in the middle of, and nil for
	// every server that is not: one with an account, and a local preview
	// where an open studio is nobody else's business.
	Setup *setup.Flow
}

// Server answers API requests.
type Server struct {
	src   SiteSource
	log   *slog.Logger
	auth  *auth.Guard
	setup *setup.Flow
}

// Routes lists the endpoints this server registers, for tests that check the
// published description against what is actually served.
func (s *Server) Routes() []string {
	out := make([]string, 0, len(s.routes()))
	for _, rt := range s.routes() {
		out = append(out, rt.Method+" "+rt.Path)
	}
	return out
}

// New returns a server reading through src.
func New(opts Options) *Server {
	log := opts.Logger
	if log == nil {
		log = discardLogger()
	}
	guard := opts.Auth
	if guard == nil {
		// An open server still gets a guard, because the cross-origin check
		// is not about who is calling: a page on the internet can post to a
		// server listening on localhost, and an admin with no password is
		// exactly the one that would do as it was told.
		guard = auth.New(nil)
	}
	return &Server{src: opts.Site, log: log, auth: guard, setup: opts.Setup}
}

// Handler returns the API's own routes, addressed without [Prefix].
//
// The routes live on a mux of their own so that the prefix is stripped before
// matching. Putting them on a shared mux next to the site's catch-all would
// let the catch-all answer first, and every API request that missed would come
// back as one of the site's HTML pages.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Patterns carry their method, so a write gets 405 from the router rather
	// than from a handler that remembered to check.
	for _, rt := range s.routes() {
		mux.HandleFunc(rt.Method+" "+rt.Path, rt.Handler)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Who is asking is settled before the router is consulted, so that a
		// caller with no session cannot learn which endpoints exist by
		// reading which ones answer 404.
		if !s.allow(w, r) {
			return
		}

		// An empty pattern means the router itself matched nothing, so no
		// handler of ours will run and whatever is written is the router's
		// own refusal. Anything else is a handler's answer and must reach the
		// client untouched: a handler's "no such content" is not the router's
		// "no such endpoint".
		if _, pattern := mux.Handler(r); pattern == "" {
			mux.ServeHTTP(&jsonErrors{ResponseWriter: w, req: r}, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// jsonErrors restates the router's own plain-text refusals in the API's error
// shape.
//
// A catch-all route would be the obvious way to do this, but a catch-all
// matches every method, which takes away the router's chance to distinguish
// "this endpoint does not exist" from "this endpoint is read-only" -- exactly
// the distinction a client needs when writes arrive alongside reads.
type jsonErrors struct {
	http.ResponseWriter
	req     *http.Request
	replied bool
}

func (w *jsonErrors) WriteHeader(status int) {
	switch status {
	case http.StatusNotFound:
		w.replied = true
		fail(w.ResponseWriter, status, CodeNotFound, "no such endpoint: "+Prefix+w.req.URL.Path)
	case http.StatusMethodNotAllowed:
		w.replied = true
		fail(w.ResponseWriter, status, CodeInvalidRequest,
			w.req.Method+" is not allowed on "+Prefix+w.req.URL.Path)
	default:
		w.ResponseWriter.WriteHeader(status)
	}
}

// Write drops the router's own message once this wrapper has answered.
func (w *jsonErrors) Write(b []byte) (int, error) {
	if w.replied {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

// Mount registers the API under [Prefix] on an existing mux.
func (s *Server) Mount(mux *http.ServeMux) {
	mux.Handle(Prefix+"/", http.StripPrefix(Prefix, s.Handler()))
}

// route is one endpoint.
type route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// routes is the table the router and the OpenAPI document are both built
// from, so that an endpoint cannot exist without being described or be
// described without existing.
func (s *Server) routes() []route {
	return []route{
		{http.MethodGet, OpenAPIPath, s.handleOpenAPI},
		{http.MethodGet, "/setup", s.handleSetupState},
		{http.MethodPost, "/setup", s.handleSetup},
		{http.MethodGet, "/auth/session", s.handleSession},
		{http.MethodPost, "/auth/login", s.handleLogin},
		{http.MethodPost, "/auth/logout", s.handleLogout},
		{http.MethodGet, "/site", s.handleSite},
		{http.MethodGet, "/content-types", s.handleContentTypes},
		{http.MethodGet, "/publish", s.handlePublishState},
		{http.MethodPost, "/publish", s.handlePublish},
		{http.MethodPost, "/publish/preflight", s.handlePreflight},
		{http.MethodPost, "/publish/push", s.handlePush},
		{http.MethodGet, "/settings", s.handleSettings},
		{http.MethodPut, "/settings", s.handleUpdateSettings},
		{http.MethodGet, "/contents", s.handleContents},
		{http.MethodPost, "/contents", s.handleCreate},
		{http.MethodGet, "/contents/{id}", s.handleContent},
		{http.MethodPut, "/contents/{id}", s.handleUpdate},
		{http.MethodDelete, "/contents/{id}", s.handleDelete},
		{http.MethodPost, "/contents/{id}/restore", s.handleRestore},
		{http.MethodPost, "/preview", s.handlePreview},
		{http.MethodPost, "/wordcount", s.handleWordCount},
		{http.MethodPost, "/contents/{id}/media", s.handleUpload},
		{http.MethodDelete, "/contents/{id}/media/{name}", s.handleDeleteMedia},
		{http.MethodGet, "/taxonomies", s.handleTaxonomies},
		{http.MethodGet, "/taxonomies/{taxonomy}/terms", s.handleTerms},
		{http.MethodGet, "/taxonomies/{taxonomy}/terms/{term}", s.handleTerm},
		{http.MethodPut, "/taxonomies/{taxonomy}/terms/{term}", s.handleRenameTerm},
		{http.MethodDelete, "/taxonomies/{taxonomy}/terms/{term}", s.handleRemoveTerm},
		{http.MethodPost, "/media", s.handleUploadSiteMedia},
		{http.MethodGet, "/themes", s.handleThemes},
		{http.MethodPost, "/themes", s.handleInstallTheme},
		{http.MethodGet, "/themes/{name}", s.handleTheme},
		{http.MethodDelete, "/themes/{name}", s.handleRemoveTheme},
		{http.MethodGet, "/themes/{name}/screenshot", s.handleThemeScreenshot},
	}
}
