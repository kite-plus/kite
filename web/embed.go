// Package web carries the admin, built by pnpm and compiled into the binary.
//
// It is embedded rather than downloaded or served from disk because the admin
// and the API it calls are one program: a version of either that has drifted
// from the other is a class of bug nobody should have to debug.
package web

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Path is where the admin is mounted.
const Path = "/admin"

// dist is the built admin. The directory is committed empty so that `go build`
// works on a machine with no Node; a build that skipped pnpm therefore
// compiles and reports that the admin is missing rather than failing to link.
//
//go:embed all:dist
var dist embed.FS

// Built reports whether this binary carries an admin.
func Built() bool {
	_, err := fs.Stat(dist, "dist/index.html")
	return err == nil
}

// Handler serves the admin under [Path].
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil || !Built() {
		return http.HandlerFunc(notBuilt)
	}
	files := http.FileServerFS(sub)

	return http.StripPrefix(Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}

		// A path with no file behind it is a route inside the admin, not a
		// missing asset: the client router has to be given the page so it can
		// resolve the route itself. An asset request is not rewritten, or a
		// mistyped script URL would return HTML and fail far from its cause.
		if _, err := fs.Stat(sub, name); errors.Is(err, fs.ErrNotExist) {
			if path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}

		// The bundle names its files by content hash, so they can be cached
		// forever; index.html names them and must not be.
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-store")
		}
		files.ServeHTTP(w, r)
	}))
}

func notBuilt(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("This build does not include the admin.\n\n" +
		"Build it with:  make web\n"))
}
