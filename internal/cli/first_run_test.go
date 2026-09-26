package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/config"
)

// runIn starts kite run in dir on a free port and returns its address and a
// channel with what the command returned, stopping it when the test ends.
func runIn(t *testing.T, dir string) (string, <-chan error) {
	t.Helper()
	t.Chdir(dir)
	addr, err := resolveAddr("", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(addr)

	cmd := newRunCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--port", port, "--open=false", "--watch=false", "--live-reload=false", "--quiet"})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		done <- cmd.ExecuteContext(ctx)
		close(finished)
	}()
	// Waited for, not only stopped: the server holds the index open, and
	// Windows will not remove the folder of a file that is still open.
	t.Cleanup(func() {
		cancel()
		<-finished
	})
	return "http://" + addr, done
}

// eventually polls until check passes or ten seconds have gone.
func eventually(t *testing.T, what string, check func() bool) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		if check() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func getJSON(url string, into any) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(into)
}

// Somebody who will not use a terminal starts kite run in an empty folder and
// finishes in a browser: the site is described there, written here, and
// served without a restart.
func TestAnEmptyFolderGetsItsSiteFromTheBrowser(t *testing.T) {
	dir := t.TempDir()
	base, done := runIn(t, dir)

	var state api.SetupState
	eventually(t, "the setup page", func() bool {
		status, err := getJSON(base+api.Prefix+"/setup", &state)
		return err == nil && status == http.StatusOK
	})
	if !state.Required || !state.NewSite {
		t.Fatalf("setup = %+v, want a new site to create", state)
	}
	var refused api.ErrorBody
	if status, _ := getJSON(base+api.Prefix+"/site", &refused); status != http.StatusServiceUnavailable ||
		refused.Error.Code != api.CodeSetupRequired {
		t.Errorf("GET /site before the site exists: status %d, %+v", status, refused.Error)
	}

	body := `{"site":{"title":"Notes","base_url":"https://notes.example.com","language":"zh-CN"}}`
	resp, err := http.Post(base+api.Prefix+"/setup", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("creating the site: status %d", resp.StatusCode)
	}

	var site api.SiteInfo
	eventually(t, "the site to be served", func() bool {
		status, err := getJSON(base+api.Prefix+"/site", &site)
		return err == nil && status == http.StatusOK
	})
	if site.Title != "Notes" || site.Language != "zh-CN" {
		t.Errorf("served site = %q in %q, want Notes in zh-CN", site.Title, site.Language)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("the project was not written: %v", err)
	}
	if cfg.Site.BaseURL != "https://notes.example.com" {
		t.Errorf("baseURL = %q", cfg.Site.BaseURL)
	}

	select {
	case err := <-done:
		t.Fatalf("kite run stopped after creating the site: %v", err)
	default:
	}
}

// A folder with anything in it is somebody's, and a site is not started on
// top of it.
func TestAFolderWithFilesInItIsNotMadeASite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, done := runIn(t, dir)

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "empty folder") {
			t.Errorf("kite run = %v, want a refusal pointing at an empty folder", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("kite run kept going in a folder that was not empty")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the folder holds %d entries, want only the file that was there", len(entries))
	}
	if _, err := os.Stat(filepath.Join(dir, "kite.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a project was written into a folder with files in it")
	}
}
