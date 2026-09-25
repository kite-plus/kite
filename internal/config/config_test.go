package config_test

import (
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
