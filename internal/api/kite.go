package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/kite-plus/kite/internal/content"
	"github.com/kite-plus/kite/internal/kitew"
)

// kiteRelease says which Kite release builds the site.
func kiteRelease(view View) KiteRelease {
	var k KiteRelease
	k.Running, _ = kitew.Release(view.Version)
	if view.Lock != nil {
		if f, err := view.Lock(); err == nil && f.Kite != nil {
			k.Pinned, k.Checksums = f.Kite.Version, f.Kite.Checksums != ""
		}
	}
	if view.Kitew != nil {
		k.Wrapper, k.Deploy = view.Kitew()
	}
	return k
}

// handlePin pins this server's release in kite.lock, so that kitew, and the
// deploy through it, build the site with the Kite it is previewed with.
func (s *Server) handlePin(w http.ResponseWriter, r *http.Request) {
	view := s.src()
	if !s.writable(w, view) {
		return
	}
	running, ok := kitew.Release(view.Version)
	if !ok {
		fail(w, http.StatusConflict, CodeConflict,
			"this Kite, "+view.Version+", is a build from source, which no release stands for")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	pin, err := kitew.Pin(ctx, http.DefaultClient, running)
	if errors.Is(err, kitew.ErrNoRelease) {
		fail(w, http.StatusConflict, CodeNoRelease, err.Error())
		return
	}
	// Any other failure pins the release without its checksums, and kitew
	// checks the download against the list the release serves.
	if _, err := view.Writer.Apply(r.Context(), content.ChangeSet{
		Ops:     []content.Op{content.PinKite{Version: pin.Version, Checksums: pin.Checksums}},
		Message: "kite: build with " + pin.Version,
	}); err != nil {
		s.failErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, kiteRelease(view))
}
