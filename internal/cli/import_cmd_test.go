package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/config"
)

const hexoSite = "../importer/hexo/testdata/site"

func runImport(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newImportHexoCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(append(args, "--workflow=false"))
	err := cmd.Execute()
	return out.String(), err
}

// An empty folder becomes a site named, addressed and dated as the Hexo site
// was, holding its content.
func TestImportingAHexoSiteMakesAProject(t *testing.T) {
	root := filepath.Join(t.TempDir(), "blog")
	out, err := runImport(t, hexoSite, root)
	if err != nil {
		t.Fatalf("import: %v\n%s", err, out)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("the new project does not load: %v", err)
	}
	if cfg.Site.Title != "旧博客" || cfg.Site.BaseURL != "https://example.com/blog" ||
		cfg.Site.Language != "zh-CN" || cfg.Site.Timezone != "Asia/Shanghai" || cfg.Site.Author != "Amy" {
		t.Errorf("site = %+v", cfg.Site)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "posts", "hello-world", "index.md")); err != nil {
		t.Errorf("the posts did not come along: %v", err)
	}
	for _, want := range []string{"posts    2, and 1 draft(s)", "content/posts/hello-world", "source/_posts/loose.png"} {
		if !strings.Contains(out, want) {
			t.Errorf("the report lacks %q:\n%s", want, out)
		}
	}
}

// Nothing is imported over files that are not a Kite project's, nor into the
// Hexo site itself.
func TestAnImportGoesOnlyWhereItBelongs(t *testing.T) {
	busy := t.TempDir()
	if err := os.WriteFile(filepath.Join(busy, "notes.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runImport(t, hexoSite, busy); err == nil || !strings.Contains(err.Error(), "not a Kite project") {
		t.Errorf("into a busy folder: %v", err)
	}
	if _, err := runImport(t, hexoSite, filepath.Join(hexoSite, "kite")); err == nil || !strings.Contains(err.Error(), "outside the Hexo site") {
		t.Errorf("into the Hexo site: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hexoSite, "kite")); err == nil {
		t.Error("a folder was made inside the Hexo site")
	}
}
