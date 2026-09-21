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
)

// Prefix is where the API is mounted. It is versioned in the path so that a
// breaking change can be introduced alongside, never in place.
const Prefix = "/api/v1"

// Options configures a server.
type Options struct {
	Site   SiteSource
	Logger *slog.Logger
}

// Server answers API requests.
type Server struct {
	src SiteSource
	log *slog.Logger
}

// New returns a server reading through src.
func New(opts Options) *Server {
	log := opts.Logger
	if log == nil {
		log = discardLogger()
	}
	return &Server{src: opts.Site, log: log}
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
	mux.HandleFunc("GET /site", s.handleSite)
	mux.HandleFunc("GET /content-types", s.handleContentTypes)
	mux.HandleFunc("GET /contents", s.handleContents)
	mux.HandleFunc("GET /contents/{id}", s.handleContent)
	mux.HandleFunc("GET /taxonomies", s.handleTaxonomies)
	mux.HandleFunc("GET /taxonomies/{taxonomy}/terms", s.handleTerms)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
