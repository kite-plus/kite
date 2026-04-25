package service

import "testing"

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{"identical", "v1.2.3", "v1.2.3", false},
		{"patch bump", "v1.2.4", "v1.2.3", true},
		{"minor bump", "v1.3.0", "v1.2.9", true},
		{"major bump", "v2.0.0", "v1.99.99", true},
		{"older latest", "v1.2.3", "v1.2.4", false},
		{"missing v prefix on latest", "1.2.3", "v1.2.2", true},
		{"missing v prefix on current", "v1.2.3", "1.2.2", true},
		{"dev current", "v1.0.0", "dev", false},
		{"dirty current", "v1.0.0", "v1.0.0-5-gabc123-dirty", false},
		{"prerelease latest is rejected", "v1.2.3-rc1", "v1.0.0", false},
		{"empty latest", "", "v1.0.0", false},
		{"empty current", "v1.0.0", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isNewerVersion(tc.latest, tc.current)
			if got != tc.want {
				t.Fatalf("isNewerVersion(%q, %q) = %v, want %v", tc.latest, tc.current, got, tc.want)
			}
		})
	}
}
