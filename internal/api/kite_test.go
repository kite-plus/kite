package api_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/kitew"
	"github.com/kite-plus/kite/internal/lock"
)

// withRelease serves the site as Kite version would, kite.lock and kitew
// read from root.
func withRelease(root, version string) func(*api.Options) {
	return func(o *api.Options) {
		inner := o.Site
		o.Site = func() api.View {
			v := inner()
			v.Version = version
			v.Lock = func() (*lock.File, error) { return lock.Read(root) }
			v.Kitew = func() (bool, string) { return kitew.Installed(root), kitew.Deploy(root) }
			return v
		}
	}
}

func kiteOf(t *testing.T, h http.Handler) api.KiteRelease {
	t.Helper()
	rec := call(t, h, http.MethodGet, api.Prefix+"/site", nil)
	var info api.SiteInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	return info.Kite
}

func TestTheStudioPinsTheReleaseItRuns(t *testing.T) {
	list := strings.Repeat("a", 64) + "  kite_0.1.7_linux_amd64.tar.gz\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0.1.7/checksums.txt" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(list))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KITE_DOWNLOAD_URL", srv.URL)
	root := newProject(t, 1)
	if _, err := kitew.Write(root); err != nil {
		t.Fatal(err)
	}

	h, _ := newWritableServer(t, root, withRelease(root, "0.1.7"))
	if k := kiteOf(t, h); k.Pinned != "" || k.Running != "0.1.7" || !k.Wrapper || k.Deploy != "" {
		t.Fatalf("before pinning: %+v", k)
	}
	rec := call(t, h, http.MethodPost, api.Prefix+"/site/pin", nil)
	var pinned api.KiteRelease
	if err := json.Unmarshal(rec.Body.Bytes(), &pinned); rec.Code != http.StatusOK || err != nil ||
		pinned.Pinned != "0.1.7" || !pinned.Checksums {
		t.Fatalf("pin: %d %s", rec.Code, rec.Body)
	}
	f, err := lock.Read(root)
	sum := sha256.Sum256([]byte(list))
	if err != nil || f.Kite == nil || f.Kite.Checksums != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Errorf("kite.lock pins %+v, %v", f.Kite, err)
	}

	// A release the downloads do not have would break every build.
	other, _ := newWritableServer(t, root, withRelease(root, "0.1.8"))
	if rec := call(t, other, http.MethodPost, api.Prefix+"/site/pin", nil); rec.Code != http.StatusConflict ||
		!strings.Contains(rec.Body.String(), api.CodeNoRelease) {
		t.Errorf("an unreleased version: %d %s", rec.Code, rec.Body)
	}
	source, _ := newWritableServer(t, root, withRelease(root, "dev"))
	if k := kiteOf(t, source); k.Running != "" || k.Pinned != "0.1.7" {
		t.Errorf("a build from source sees %+v", k)
	}
	if rec := call(t, source, http.MethodPost, api.Prefix+"/site/pin", nil); rec.Code != http.StatusConflict {
		t.Errorf("a build from source pinned itself: %d %s", rec.Code, rec.Body)
	}
	if f, _ := lock.Read(root); f.Kite.Version != "0.1.7" {
		t.Errorf("a refused pin changed kite.lock to %+v", f.Kite)
	}
}
