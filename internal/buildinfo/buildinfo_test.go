package buildinfo_test

import (
	"testing"

	"github.com/kite-plus/kite/internal/buildinfo"
)

// A release credits itself as it is released, without the v of its tag, and
// a build from source with no version says only Kite.
func TestTheGeneratorNamesTheVersion(t *testing.T) {
	was := buildinfo.Version
	t.Cleanup(func() { buildinfo.Version = was })

	for version, want := range map[string]string{
		"dev":                     "Kite",
		"":                        "Kite",
		"0.1.9":                   "Kite 0.1.9",
		"v0.1.9":                  "Kite 0.1.9",
		"v0.1.8-3-g4d43edb-dirty": "Kite 0.1.8-3-g4d43edb-dirty",
	} {
		buildinfo.Version = version
		if got := buildinfo.Generator(); got != want {
			t.Errorf("version %q: generator = %q, want %q", version, got, want)
		}
	}
}
