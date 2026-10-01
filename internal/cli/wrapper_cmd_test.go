package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/buildinfo"
	"github.com/kite-plus/kite/internal/lock"
)

// fakeReleases serves the checksums.txt of each version as kitew would
// fetch it, and returns the pin each makes.
func fakeReleases(t *testing.T, versions ...string) map[string]string {
	t.Helper()
	lists := make(map[string]string)
	pins := make(map[string]string)
	for _, v := range versions {
		list := strings.Repeat("a", 64) + "  kite_" + v + "_linux_amd64.tar.gz\n"
		lists["/v"+v+"/checksums.txt"] = list
		sum := sha256.Sum256([]byte(list))
		pins[v] = "sha256:" + hex.EncodeToString(sum[:])
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		list, ok := lists[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(list))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KITE_DOWNLOAD_URL", srv.URL)
	return pins
}

// asRelease makes the running Kite the release v, as the release workflow
// stamps one.
func asRelease(t *testing.T, v string) {
	t.Helper()
	old := buildinfo.Version
	buildinfo.Version = v
	t.Cleanup(func() { buildinfo.Version = old })
}

func readPin(t *testing.T, root string) *lock.Pin {
	t.Helper()
	f, err := lock.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	return f.Kite
}

func TestANewSiteIsPinnedToTheReleaseThatMadeIt(t *testing.T) {
	pins := fakeReleases(t, "0.1.6", "0.1.7")
	asRelease(t, "0.1.7")
	root := newSite(t)
	if pin := readPin(t, root); pin == nil || pin.Version != "0.1.7" || pin.Checksums != pins["0.1.7"] {
		t.Fatalf("a new site pins %+v", pin)
	}
	if info, err := os.Stat(filepath.Join(root, "kitew")); err != nil || (info.Mode()&0o111 == 0 && runtime.GOOS != "windows") {
		t.Errorf("kitew: %v, %v", info, err)
	}
	if out := runKite(t, root, "doctor"); strings.Contains(out, "Kite release") {
		t.Errorf("a new site's pin is reported:\n%s", out)
	}

	// Moved back a release, the pin is what the deploy builds with, and both
	// the build and the doctor say this Kite is another.
	if out := runKite(t, root, "wrapper", "--version", "v0.1.6"); !strings.Contains(out, "pinned Kite 0.1.6") {
		t.Errorf("wrapper said:\n%s", out)
	}
	if pin := readPin(t, root); pin.Version != "0.1.6" || pin.Checksums != pins["0.1.6"] {
		t.Errorf("pinned %+v", pin)
	}
	if out := runKite(t, root, "doctor"); !strings.Contains(out, "pins Kite 0.1.6 and this is 0.1.7") {
		t.Errorf("doctor said:\n%s", out)
	}
	if out := runKite(t, root, "build"); !strings.Contains(out, "note: kite.lock pins Kite 0.1.6") {
		t.Errorf("build said:\n%s", out)
	}
	runKite(t, root, "wrapper")
	if pin := readPin(t, root); pin.Version != "0.1.7" {
		t.Errorf("wrapper with no version pinned %+v", pin)
	}

	if err := runKiteErr(t, root, "wrapper", "--version", "9.9.9"); !strings.Contains(err.Error(), "never released") {
		t.Errorf("a release never made: %v", err)
	}
	if err := runKiteErr(t, root, "wrapper", "--version", "latest"); !strings.Contains(err.Error(), "names no release") {
		t.Errorf("a version that is no release: %v", err)
	}
	if pin := readPin(t, root); pin.Version != "0.1.7" {
		t.Errorf("a refused pin changed it to %+v", pin)
	}
}

// Without the network a site is still made, pinned without the checksums,
// and the doctor says how to add them.
func TestASiteMadeOfflineIsPinnedWithoutItsChecksums(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	t.Setenv("KITE_DOWNLOAD_URL", srv.URL)
	asRelease(t, "0.1.7")
	root := newSite(t)
	if pin := readPin(t, root); pin == nil || pin.Version != "0.1.7" || pin.Checksums != "" {
		t.Fatalf("pinned %+v", pin)
	}
	if out := runKite(t, root, "doctor"); !strings.Contains(out, "without the checksums") {
		t.Errorf("doctor said:\n%s", out)
	}
}

// A build from source is no release, so a site it makes pins none and says
// so, and the wrapper asks for the release to pin.
func TestABuildFromSourcePinsNoRelease(t *testing.T) {
	pins := fakeReleases(t, "0.1.6")
	asRelease(t, "v0.1.6-3-gabc1234")
	work := t.TempDir()
	out := runKite(t, work, "init", "--yes", "--workflow", "site")
	root := filepath.Join(work, "site")
	if !strings.Contains(out, "build from source") || strings.Contains(out, "which ./kitew runs") {
		t.Errorf("init said:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, lock.Name)); !os.IsNotExist(err) {
		t.Errorf("a build from source wrote a lock: %v", err)
	}
	if out := runKite(t, root, "doctor"); !strings.Contains(out, "pins no release for it to run") {
		t.Errorf("doctor said:\n%s", out)
	}
	if err := runKiteErr(t, root, "wrapper"); !strings.Contains(err.Error(), "build from source") {
		t.Errorf("wrapper without a version: %v", err)
	}
	runKite(t, root, "wrapper", "--version", "0.1.6")
	if pin := readPin(t, root); pin == nil || pin.Checksums != pins["0.1.6"] {
		t.Errorf("pinned %+v", pin)
	}
}

// A site made before kitew deploys with the Kite its workflow installs, and
// the wrapper says what to change rather than editing the author's file.
func TestAnOlderDeployWorkflowIsPointedAtKitew(t *testing.T) {
	fakeReleases(t, "0.1.7")
	asRelease(t, "0.1.7")
	root := newSite(t)
	old := "jobs:\n  build:\n    steps:\n      - run: go install github.com/kite-plus/kite/cmd/kite@v0.1.5\n      - run: kite build\n"
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(WorkflowPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, WorkflowPath), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	if out := runKite(t, root, "wrapper"); !strings.Contains(out, "installs Kite on its own") {
		t.Errorf("wrapper said:\n%s", out)
	}
	if data, _ := os.ReadFile(filepath.Join(root, WorkflowPath)); string(data) != old {
		t.Errorf("the workflow was changed:\n%s", data)
	}
	if out := runKite(t, root, "doctor"); !strings.Contains(out, "rather than running kitew") {
		t.Errorf("doctor said:\n%s", out)
	}
}
