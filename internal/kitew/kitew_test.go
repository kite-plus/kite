package kitew

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

// TestMain lets the test binary stand in for a released kite: kitew runs it
// with KITEW_TEST_KITE set, and it says what it was asked to do.
func TestMain(m *testing.M) {
	if os.Getenv("KITEW_TEST_KITE") != "" {
		fmt.Printf("kite %s\n", strings.Join(os.Args[1:], " "))
		os.Exit(3)
	}
	os.Exit(m.Run())
}

func TestAReleaseIsNamedByItsTag(t *testing.T) {
	for v, want := range map[string]string{"0.1.6": "0.1.6", "v0.1.6": "0.1.6", "v0.2.0-rc.1": "0.2.0-rc.1"} {
		if got, ok := Release(v); !ok || got != want {
			t.Errorf("Release(%q) = %q, %v", v, got, ok)
		}
	}
	for _, v := range []string{"dev", "v0.1.6-3-gabc1234", "v0.1.6-dirty", "26200b0", "577feeb-dirty", "0.1.1-snapshot-abc"} {
		if _, ok := Release(v); ok {
			t.Errorf("Release(%q) names a release", v)
		}
	}
}

func TestThePinRecordsTheChecksumsItWasMadeWith(t *testing.T) {
	list := "1111  kite_0.1.7_linux_amd64.tar.gz\n2222  kite_0.1.7_windows_amd64.zip\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v0.1.7/checksums.txt":
			_, _ = w.Write([]byte(list))
		case "/v0.1.8/checksums.txt":
			_, _ = w.Write([]byte("<html>a mirror's error page</html>"))
		case "/v0.1.9/checksums.txt":
			http.Error(w, "busy", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KITE_DOWNLOAD_URL", srv.URL+"/")

	pin, err := Pin(t.Context(), srv.Client(), "0.1.7")
	sum := sha256.Sum256([]byte(list))
	if err != nil || pin.Version != "0.1.7" || pin.Checksums != "sha256:"+hex.EncodeToString(sum[:]) {
		t.Errorf("Pin = %+v, %v", pin, err)
	}
	if _, err := Pin(t.Context(), srv.Client(), "9.9.9"); !errors.Is(err, ErrNoRelease) {
		t.Errorf("a version never released: %v", err)
	}
	for _, v := range []string{"0.1.8", "0.1.9"} {
		if pin, err := Pin(t.Context(), srv.Client(), v); err == nil || errors.Is(err, ErrNoRelease) ||
			pin.Version != v || pin.Checksums != "" {
			t.Errorf("Pin(%s) = %+v, %v", v, pin, err)
		}
	}
}

func TestWriteLeavesTheScriptsReadyToRun(t *testing.T) {
	root := t.TempDir()
	if changed, err := Write(root); err != nil || strings.Join(changed, " ") != "kitew kitew.ps1" {
		t.Fatalf("Write = %v, %v", changed, err)
	}
	if !Installed(root) {
		t.Error("the scripts written are not seen")
	}
	if changed, err := Write(root); err != nil || len(changed) != 0 {
		t.Errorf("writing them again changed %v, %v", changed, err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	if info, err := os.Stat(filepath.Join(root, "kitew")); err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("kitew is %v, %v", info.Mode(), err)
	}
	// A checkout that lost the bit, as one made on Windows does, gets it back.
	if err := os.Chmod(filepath.Join(root, "kitew"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, err := Write(root); err != nil || strings.Join(changed, " ") != "kitew" {
		t.Errorf("Write over a kitew that is not executable = %v, %v", changed, err)
	}
}

// fakeRelease serves a release of Kite for this machine, its kite being the
// test binary, and counts the archives fetched.
type fakeRelease struct {
	srv      *httptest.Server
	files    map[string][]byte
	archives atomic.Int32
}

func newFakeRelease(t *testing.T) *fakeRelease {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	arch := runtime.GOARCH
	if arch == "arm" {
		arch = "armv7"
	}
	var name string
	var archive []byte
	if runtime.GOOS == "windows" {
		name, archive = "kite_0.1.7_windows_"+arch+".zip", zipOf(t, "kite.exe", bin)
	} else {
		name, archive = "kite_0.1.7_"+runtime.GOOS+"_"+arch+".tar.gz", tarOf(t, "kite", bin)
	}
	sum := sha256.Sum256(archive)
	list := hex.EncodeToString(sum[:]) + "  " + name + "\n" + strings.Repeat("0", 64) + "  kite_0.1.7_plan9_amd64.tar.gz\n"
	// 0.1.8 lists a checksum its archive does not have.
	wrong := strings.Repeat("0", 64) + "  " + strings.ReplaceAll(name, "0.1.7", "0.1.8") + "\n"

	r := &fakeRelease{files: map[string][]byte{
		"/v0.1.7/checksums.txt": []byte(list),
		"/v0.1.7/" + name:       archive,
		"/v0.1.8/checksums.txt": []byte(wrong),
		"/v0.1.8/" + strings.ReplaceAll(name, "0.1.7", "0.1.8"): archive,
	}}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		data, ok := r.files[req.URL.Path]
		if !ok {
			http.NotFound(w, req)
			return
		}
		if !strings.HasSuffix(req.URL.Path, ".txt") {
			r.archives.Add(1)
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(r.srv.Close)
	return r
}

// checksums is the pin of a version's checksums.txt as kite.lock records it.
func (r *fakeRelease) checksums(version string) string {
	sum := sha256.Sum256(r.files["/v"+version+"/checksums.txt"])
	return "sha256:" + hex.EncodeToString(sum[:])
}

func tarOf(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipOf(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// wrapper is a script run the way a person or a deploy would.
type wrapper struct {
	name string
	args []string
}

func (w wrapper) run(t *testing.T, r *fakeRelease, lockFile, cache string, args ...string) (string, int) {
	t.Helper()
	root := t.TempDir()
	if _, err := Write(root); err != nil {
		t.Fatal(err)
	}
	if lockFile != "" {
		if err := os.WriteFile(filepath.Join(root, "kite.lock"), []byte(lockFile), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(w.name, append(append(append([]string{}, w.args...), filepath.Join(root, w.script())), args...)...)
	cmd.Env = append(os.Environ(),
		"KITE_DOWNLOAD_URL="+r.srv.URL,
		"KITEW_TEST_KITE=1",
		"HOME="+cache,
		"XDG_CACHE_HOME="+filepath.Join(cache, ".cache"),
		"LOCALAPPDATA="+cache,
	)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("%s: %v\n%s", w.name, err, out)
	}
	return string(out), cmd.ProcessState.ExitCode()
}

func (w wrapper) script() string {
	if w.name == "sh" {
		return "kitew"
	}
	return "kitew.ps1"
}

// wrappers are the scripts this machine can run: kitew where there is a
// POSIX shell that is not Windows', and kitew.ps1 under every PowerShell
// installed, Windows PowerShell included.
func wrappers(t *testing.T) []wrapper {
	var out []wrapper
	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("sh"); err == nil {
			out = append(out, wrapper{name: "sh"})
		}
	}
	for _, shell := range []string{"pwsh", "powershell"} {
		if _, err := exec.LookPath(shell); err != nil || (shell == "powershell" && runtime.GOOS != "windows") {
			continue
		}
		args := []string{"-NoProfile", "-NonInteractive"}
		if runtime.GOOS == "windows" {
			args = append(args, "-ExecutionPolicy", "Bypass")
		}
		out = append(out, wrapper{name: shell, args: append(args, "-File")})
	}
	if len(out) == 0 {
		t.Skip("no shell to run the scripts with")
	}
	return out
}

func TestTheWrapperRunsThePinnedRelease(t *testing.T) {
	for _, w := range wrappers(t) {
		t.Run(w.name, func(t *testing.T) {
			r := newFakeRelease(t)
			cache := t.TempDir()
			lockFile := "lockfileVersion: 1\nkite:\n  version: 0.1.7\n  checksums: " + r.checksums("0.1.7") + "\n"

			out, code := w.run(t, r, lockFile, cache, "build", "--verify")
			if code != 3 || !strings.Contains(out, "kite build --verify") || !strings.Contains(out, "downloading Kite 0.1.7") {
				t.Fatalf("first run: exit %d\n%s", code, out)
			}
			// The second build runs the copy kept, without the network.
			out, code = w.run(t, r, lockFile, cache, "version")
			if code != 3 || !strings.Contains(out, "kite version") || strings.Contains(out, "downloading") || r.archives.Load() != 1 {
				t.Errorf("second run: exit %d, %d archives fetched\n%s", code, r.archives.Load(), out)
			}

			// A pin made before the release's checksums were swapped.
			out, code = w.run(t, r, "kite:\n  version: 0.1.7\n  checksums: sha256:"+strings.Repeat("ab", 32)+"\n", t.TempDir(), "build")
			if code != 1 || !strings.Contains(out, "not the ones kite.lock pins") || strings.Contains(out, "kite build") {
				t.Errorf("swapped checksums: exit %d\n%s", code, out)
			}
			// An archive its checksum does not match, from a pin without one.
			out, code = w.run(t, r, "kite:\n  version: 0.1.8\n", t.TempDir(), "build")
			if code != 1 || !strings.Contains(out, "does not match its checksum") {
				t.Errorf("a wrong archive: exit %d\n%s", code, out)
			}
			out, code = w.run(t, r, "lockfileVersion: 1\nthemes: {}\n", t.TempDir(), "build")
			if code != 1 || !strings.Contains(out, "pins no Kite release") {
				t.Errorf("no pin: exit %d\n%s", code, out)
			}
			out, code = w.run(t, r, "", t.TempDir(), "build")
			if code != 1 || !strings.Contains(out, "no kite.lock") {
				t.Errorf("no lock: exit %d\n%s", code, out)
			}
		})
	}
}
