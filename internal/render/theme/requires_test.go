package theme

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kite-plus/kite/internal/buildinfo"
)

func TestARequiresRangeAdmitsTheVersionsItNames(t *testing.T) {
	for _, tc := range []struct {
		requires, kite string
		ok             bool
	}{
		{">=1.0.0 <2.0.0", "v1.4.2", true},
		{">=1.0.0 <2.0.0", "v2.0.0", false},
		{">=1.0.0 <2.0.0", "v0.9.9", false},
		{">=1.0.0, <2.0.0", "1.0.0", true},
		{">=1.2", "v1.2.0", true},
		{">1.2", "v1.2.0", false},
		{"<=1", "v1.0.0", true},
		{"1.3.0", "v1.3.0", true},
		{"=1.3.0", "v1.3.1", false},
		{"<1 || >=3", "v3.1.0", true},
		{"<1 || >=3", "v2.9.0", false},
		// A pre-release comes before the release it leads up to.
		{">=1.0.0", "v1.0.0-rc.1", false},
		{">=1.0.0-rc.2", "v1.0.0-rc.10", true},
		{">=1.0.0-rc.2", "v1.0.0-rc.1", false},
		{">=1.0.0-alpha", "v1.0.0-1", false},
		{">=1.0.0", "v1.0.0+build.7", true},
	} {
		r, err := parseRange(tc.requires)
		if err != nil {
			t.Errorf("parseRange(%q): %v", tc.requires, err)
			continue
		}
		v, err := parseVersion(tc.kite)
		if err != nil {
			t.Errorf("parseVersion(%q): %v", tc.kite, err)
			continue
		}
		if got := r.admits(v); got != tc.ok {
			t.Errorf("%q admits %s = %v, want %v", tc.requires, tc.kite, got, tc.ok)
		}
	}
}

func TestAMalformedRangeIsReported(t *testing.T) {
	for _, bad := range []string{">=one", ">=1.2.3.4", "||", ">=1.0.0-", ">= "} {
		m := Manifest{Name: "paper", Requires: bad}
		if err := m.Satisfied("v1.0.0"); err == nil || !strings.Contains(err.Error(), "not a version range") {
			t.Errorf("requires %q: err = %v", bad, err)
		}
	}
}

// A theme made for another Kite is refused when it loads, saying what it
// needs and what is running, rather than failing somewhere in a template.
func TestAThemeForAnotherKiteIsRefusedWhenItLoads(t *testing.T) {
	theme := fstest.MapFS{
		"theme.yaml":          {Data: []byte("name: paper\napiVersion: kite/v1\nrequires: \">=2.0.0 <3.0.0\"\n")},
		"layouts/single.html": {Data: []byte("{{ .Page.Title }}")},
	}
	was := buildinfo.Version
	t.Cleanup(func() { buildinfo.Version = was })

	buildinfo.Version = "v1.4.0"
	_, err := Load(theme)
	if err == nil {
		t.Fatal("a theme requiring Kite 2 loaded on Kite 1.4")
	}
	for _, want := range []string{"paper", ">=2.0.0 <3.0.0", "v1.4.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not mention %q: %v", want, err)
		}
	}

	buildinfo.Version = "v2.1.0"
	if _, err := Load(theme); err != nil {
		t.Errorf("Kite 2.1 was refused: %v", err)
	}

	// A build from source is not a release and cannot be placed in a range.
	for _, dev := range []string{"dev", "4b3c00b", "4b3c00b-dirty", "1234567"} {
		buildinfo.Version = dev
		if _, err := Load(theme); err != nil {
			t.Errorf("development build %q was refused: %v", dev, err)
		}
	}

	// git describe marks a build made after a tag; it carries that release.
	for kite, ok := range map[string]bool{
		"v2.0.0-3-g4b3c00b":       true,
		"v2.0.0-3-g4b3c00b-dirty": true,
		"v2.0.0-dirty":            true,
		"v1.9.0-12-gabcdef0":      false,
		"v2.0.0-rc.1-2-gabcdef0":  false,
	} {
		buildinfo.Version = kite
		if _, err := Load(theme); (err == nil) != ok {
			t.Errorf("Kite %s: loaded = %v, want %v (%v)", kite, err == nil, ok, err)
		}
	}
}
