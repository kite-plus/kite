package api

import (
	"maps"
	"slices"
	"testing"
)

// listParameters is derived from knownParams, so the two cannot list different
// names. What can go wrong is a parameter gaining acceptance without gaining
// a sentence a client can read, which is what this checks -- along with the
// reverse, a description for something the parser would refuse.
//
// Both lists are read here rather than copied: a test that repeats a list only
// ever checks its own copy.
func TestEveryAcceptedParameterIsExplained(t *testing.T) {
	described := make([]string, 0, len(knownParams))
	for _, p := range listParameters() {
		described = append(described, p.Name)
		if p.Description == "" {
			t.Errorf("?%s is accepted but has no description", p.Name)
		}
	}

	if want := slices.Sorted(slices.Values(knownParams)); !slices.Equal(want, slices.Sorted(slices.Values(described))) {
		t.Errorf("described parameters do not match the accepted ones\n accepted: %v\ndescribed: %v",
			want, described)
	}

	for name := range maps.Keys(paramDoc) {
		if !slices.Contains(knownParams, name) {
			t.Errorf("?%s is documented but the parser does not accept it", name)
		}
	}
}
