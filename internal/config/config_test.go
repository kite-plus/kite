package config_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/config"
)

func load(t *testing.T, yaml string) (*config.Config, error) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, config.Name), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return config.Load(root)
}

// Keywords written by hand are often one line, separated by commas in either
// script, and they read as the same list a form writes.
func TestKeywordsReadAsAListOrALine(t *testing.T) {
	for _, tc := range []struct{ name, yaml string }{
		{"a list", "keywords: [books, 写作, travel]"},
		{"a line", "keywords: books, 写作， travel"},
		{"a line with repeats and gaps", "keywords: 'books,,写作、travel, books'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := load(t, "site:\n  title: T\n  baseURL: https://example.com\n  "+tc.yaml+"\n")
			if err != nil {
				t.Fatal(err)
			}
			if want := []string{"books", "写作", "travel"}; !slices.Equal(cfg.Site.Keywords, want) {
				t.Errorf("keywords = %q, want %q", cfg.Site.Keywords, want)
			}
		})
	}
}

// A zone that does not exist, or the zone of whatever machine happens to
// build, would give the same site different dates, so either is refused when
// the configuration is read.
func TestATimeZoneMustNameAZone(t *testing.T) {
	for _, tc := range []struct {
		zone string
		ok   bool
	}{
		{"", true},
		{"UTC", true},
		{"Asia/Shanghai", true},
		{"America/New_York", true},
		{"Local", false},
		{"Mars/Olympus", false},
	} {
		cfg, err := load(t, "site:\n  title: T\n  baseURL: https://example.com\n  timezone: '"+tc.zone+"'\n")
		switch {
		case tc.ok && err != nil:
			t.Errorf("%q: %v", tc.zone, err)
		case !tc.ok && err == nil:
			t.Errorf("%q was accepted", tc.zone)
		case !tc.ok && !strings.Contains(err.Error(), "site.timezone"):
			t.Errorf("%q: the error does not name the setting: %v", tc.zone, err)
		case tc.ok && tc.zone != "":
			if loc, _ := cfg.Site.Location(); loc == nil || loc.String() != tc.zone {
				t.Errorf("%q: location = %v", tc.zone, loc)
			}
		}
	}
}

// Plugins were a plain list before they had settings, and such a file still
// reads as the plugins to run.
func TestPluginsReadAsAListOrWithTheirSettings(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"

	cfg, err := load(t, site+"plugins: [search, comments]\n")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"search", "comments"}; !slices.Equal(cfg.Plugins.Enabled, want) {
		t.Errorf("enabled = %q, want %q", cfg.Plugins.Enabled, want)
	}

	cfg, err = load(t, site+`plugins:
  enabled: [comments]
  settings:
    comments: {provider: giscus}
    search: {placeholder: Find}
`)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.Plugins.Enabled, []string{"comments"}) {
		t.Errorf("enabled = %q, want only comments", cfg.Plugins.Enabled)
	}
	// A plugin that is off keeps what it was set to.
	if cfg.Plugins.Settings["search"]["placeholder"] != "Find" || cfg.Plugins.Settings["comments"]["provider"] != "giscus" {
		t.Errorf("settings = %v", cfg.Plugins.Settings)
	}
}

func TestAPluginIsNamedByAnIdOnce(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"
	for _, list := range []string{"[Search]", "[my_plugin]", "[../x]", "[search, search]"} {
		if _, err := load(t, site+"plugins: "+list+"\n"); err == nil || !strings.Contains(err.Error(), "plugins.enabled") {
			t.Errorf("plugins: %s = %v, want a refusal naming plugins.enabled", list, err)
		}
	}
	if _, err := load(t, site+"plugins: [search, code-copy]\n"); err != nil {
		t.Errorf("valid ids were refused: %v", err)
	}
}

// An absolute output is used as it stands, from kite.yaml or KITE_BUILD_OUTPUT
// alike; a relative one is taken from the project root.
func TestAnAbsoluteOutputIsUsedAsItStands(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"
	elsewhere := filepath.Join(t.TempDir(), "out")

	for _, tc := range []struct {
		name, yaml, env, want string
	}{
		{"the default", "", "", "public"},
		{"a relative path", "build:\n  output: site/public\n", "", filepath.Join("site", "public")},
		{"an absolute path", "build:\n  output: '" + elsewhere + "'\n", "", elsewhere},
		{"an absolute path from the environment", "build:\n  output: site/public\n", elsewhere, elsewhere},
		{"an absolute path through a parent", "", filepath.Join(elsewhere, "x") + string(filepath.Separator) + "..", elsewhere},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KITE_BUILD_OUTPUT", tc.env)
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, config.Name), []byte(site+tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.want
			if !filepath.IsAbs(want) {
				want = filepath.Join(root, want)
			}
			if got := cfg.Build.OutputDir(root); got != want {
				t.Errorf("output dir = %q, want %q", got, want)
			}
		})
	}
}

// A relative output is taken from the project root, and one that climbs out
// of it is refused: building outside the project takes an absolute path.
func TestARelativeOutputStaysInsideTheProject(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"
	t.Setenv("KITE_BUILD_OUTPUT", "")
	for _, out := range []string{"../out", "public/../../out"} {
		if _, err := load(t, site+"build:\n  output: "+out+"\n"); err == nil || !strings.Contains(err.Error(), "build.output") {
			t.Errorf("output: %s = %v, want a refusal naming build.output", out, err)
		}
	}
	t.Setenv("KITE_BUILD_OUTPUT", "../out")
	if _, err := load(t, site); err == nil || !strings.Contains(err.Error(), "build.output") {
		t.Errorf("KITE_BUILD_OUTPUT=../out = %v, want a refusal naming build.output", err)
	}
}

// A site moved from Hugo keeps its subscribers by writing the feed where they
// subscribed too. Each such path is a feed's own file within the site.
func TestFeedAliasesAreFilesOfTheirOwn(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"
	cfg, err := load(t, site+"build:\n  feedAliases: [/index.xml, posts/index.xml, ' atom.xml ']\n")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := []string{"index.xml", "posts/index.xml", "atom.xml"}; !slices.Equal(cfg.Build.FeedAliases, want) {
		t.Errorf("feedAliases = %q, want %q", cfg.Build.FeedAliases, want)
	}
	for _, list := range []string{"['']", "[feed/]", "[../out.xml]", "[index.html]", "[feed]", "[rss.xml]", "[sitemap.xml]", "[a.xml, /a.xml]"} {
		if _, err := load(t, site+"build:\n  feedAliases: "+list+"\n"); err == nil || !strings.Contains(err.Error(), "build.feedAliases") {
			t.Errorf("feedAliases: %s = %v, want a refusal naming build.feedAliases", list, err)
		}
	}
}

// A site sizes the pages of each kind of listing, 0 for all on one, over what
// its theme declares; a kind it does not know or a size below 0 is refused.
func TestPaginationSizesEachKindOfListing(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"
	cfg, err := load(t, site+"build:\n  pageSize: 12\n  pagination: {home: 0, term: 30}\n")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.Build.PageSizes(map[string]int{"home": 5, "list": 0})
	if want := map[string]int{"home": 0, "list": 0, "term": 30}; !maps.Equal(got, want) {
		t.Errorf("PageSizes = %v, want %v", got, want)
	}
	if cfg.Build.PageSize != 12 {
		t.Errorf("pageSize = %d, want 12 for the kinds nothing sizes", cfg.Build.PageSize)
	}
	for _, sizes := range []string{"{taxonomy: 0}", "{single: 3}", "{home: -1}"} {
		if _, err := load(t, site+"build:\n  pagination: "+sizes+"\n"); err == nil || !strings.Contains(err.Error(), "build.pagination") {
			t.Errorf("pagination: %s = %v, want a refusal naming build.pagination", sizes, err)
		}
	}
}

// A menu is read with its links in order and the links under them, and one
// a theme could not draw, or a template could not reach, is refused on load.
func TestMenusAreCheckedWhenTheyAreRead(t *testing.T) {
	const site = "site:\n  title: T\n  baseURL: https://example.com\n"
	cfg, err := load(t, site+`menus:
  main:
    - name: About
      url: /about/
    - name: Elsewhere
      params: {icon: globe}
      children:
        - {name: Code, url: "https://github.com/example"}
  footer_links:
    - {name: Feed, url: rss.xml}
`)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	main := cfg.Menus["main"]
	if len(main) != 2 || main[0].Name != "About" || main[0].URL != "/about/" ||
		main[1].Params["icon"] != "globe" || len(main[1].Children) != 1 || main[1].Children[0].Name != "Code" {
		t.Errorf("main = %+v", main)
	}
	if len(cfg.Menus["footer_links"]) != 1 {
		t.Errorf("footer_links = %+v", cfg.Menus["footer_links"])
	}

	for _, tc := range []struct{ name, menus, says string }{
		{"a name a template cannot reach", "  footer-links:\n    - {name: A, url: /a/}\n", "not a menu name"},
		{"a link with no name", "  main:\n    - {url: /a/}\n", "main > link 1: give the link a name"},
		{"a link that leads nowhere", "  main:\n    - {name: A}\n", "main > A: give the link a url"},
		{"a menu four levels deep", "  main:\n    - name: A\n      children:\n        - name: B\n          children:\n            - name: C\n              children:\n                - {name: D, url: /d/}\n", "at most 3 levels"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := load(t, site+"menus:\n"+tc.menus); err == nil || !strings.Contains(err.Error(), tc.says) {
				t.Errorf("Load = %v, want a refusal saying %q", err, tc.says)
			}
		})
	}
}
