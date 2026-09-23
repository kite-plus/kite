package serve

import (
	"bytes"
	"context"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/build"
)

// extras are the files the completion hooks write for one plan, such as the
// feed and the sitemap.
//
// They are worked out the first time a request needs them rather than on
// every reload: that means rendering every body in the site, and most
// previews never ask for a feed.
type extras struct {
	builder *build.Builder
	plan    *build.Plan
	log     *slog.Logger

	once  sync.Once
	files map[string][]byte
}

func (e *extras) get(ctx context.Context) map[string][]byte {
	e.once.Do(func() {
		files, err := e.builder.Extras(ctx, e.plan)
		if err != nil {
			// Said once for this plan rather than on every request; the next
			// reload tries again.
			e.log.Error("completion hooks failed", "err", err)
			return
		}
		e.files = files
	})
	return e.files
}

// serveExtra answers for a file the completion hooks write.
//
// It is asked before the static files, because a build writes these last,
// over any static file of the same name.
func (s *Server) serveExtra(w http.ResponseWriter, r *http.Request) bool {
	rel := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

	s.mu.RLock()
	current := s.extras
	s.mu.RUnlock()

	// Every request waiting on the same files needs them, so the client that
	// asked first going away must not abandon them.
	data, ok := current.get(context.WithoutCancel(r.Context()))[rel]
	if !ok {
		return false
	}
	if ctype := mime.TypeByExtension(path.Ext(rel)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, rel, time.Time{}, bytes.NewReader(data))
	return true
}
