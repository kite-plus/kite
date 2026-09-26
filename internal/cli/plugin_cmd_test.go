package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/config"
)

// runKiteErr runs a command line that is expected to fail, and returns why.
func runKiteErr(t *testing.T, root string, args ...string) error {
	t.Helper()
	t.Chdir(root)
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(t.Context())
	if err == nil {
		t.Fatalf("kite %s succeeded\n%s", strings.Join(args, " "), out.String())
	}
	return err
}

// A plugin started with kite plugin new goes into a site and onto its pages
// with the commands alone, and comes back out the same way.
func TestAPluginIsAddedEnabledBuiltAndRemovedFromTheCommandLine(t *testing.T) {
	work := t.TempDir()
	runKite(t, work, "plugin", "new", "greet")
	if out := runKite(t, work, "plugin", "verify", "greet"); !strings.Contains(out, "greet 0.1.0 loads") {
		t.Errorf("verify said:\n%s", out)
	}

	site := newSite(t)
	writePost(t, site, "01J8KQ2P3R4S5T6V7W8X9YZ000", "hello", "status: published\npublished_at: 2026-01-01T00:00:00Z\n")
	runKite(t, site, "plugin", "add", filepath.Join(work, "greet"))
	if err := runKiteErr(t, site, "plugin", "add", filepath.Join(work, "greet")); !strings.Contains(err.Error(), "--replace") {
		t.Errorf("adding it twice = %v, want a pointer to --replace", err)
	}
	if out := runKite(t, site, "plugin", "list"); !strings.Contains(out, "off  greet") {
		t.Errorf("list before enabling:\n%s", out)
	}

	runKite(t, site, "plugin", "enable", "greet")
	cfg, err := config.Load(site)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.Plugins.Enabled, []string{"greet"}) {
		t.Fatalf("enabled = %q", cfg.Plugins.Enabled)
	}
	runKite(t, site, "build")
	page, err := os.ReadFile(filepath.Join(site, "public", "posts", "hello", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<p class="greet">Thanks for reading.</p>`) {
		t.Error("the built post does not carry the plugin's line")
	}
	if _, err := os.Stat(filepath.Join(site, "public", "plugins", "greet", "greet.css")); err != nil {
		t.Errorf("the plugin's stylesheet was not published: %v", err)
	}

	if err := runKiteErr(t, site, "plugin", "remove", "greet"); !strings.Contains(err.Error(), "disable") {
		t.Errorf("removing an enabled plugin = %v, want a pointer to disable", err)
	}
	runKite(t, site, "plugin", "disable", "greet")
	runKite(t, site, "plugin", "remove", "greet")
	if _, err := os.Stat(filepath.Join(site, "plugins", "greet")); !os.IsNotExist(err) {
		t.Error("the plugin's directory is still there")
	}
}

func TestAPluginThatDoesNotLoadIsNotEnabled(t *testing.T) {
	site := newSite(t)
	dir := filepath.Join(site, "plugins", "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte("id: broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runKiteErr(t, site, "plugin", "enable", "broken"); !strings.Contains(err.Error(), "name is required") {
		t.Errorf("enable = %v, want the plugin's own problem", err)
	}
	if out := runKite(t, site, "plugin", "list"); !strings.Contains(out, "name is required") {
		t.Errorf("list does not say why it cannot load:\n%s", out)
	}
}
