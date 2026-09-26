package theme

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Satisfied checks the theme's requires range against the running Kite.
//
// A theme written for a Kite it is not running on is refused rather than
// rendered and left to fail somewhere in a template. A version that is not a
// release, such as a build from source, cannot be placed in a range, so it
// is not held to one.
func (m *Manifest) Satisfied(kite string) error {
	return Requires("theme", m.Name, m.Requires, kite)
}

// Requires checks the range of Kite versions an extension says it needs
// against the running Kite. kind and name say what the extension is, as
// "plugin" and "search", for the message that refuses it.
func Requires(kind, name, requires, kite string) error {
	if requires == "" {
		return nil
	}
	want, err := parseRange(requires)
	if err != nil {
		return fmt.Errorf("%s %s: requires %q is not a version range: %w", kind, name, requires, err)
	}
	// A short commit hash made only of digits would read as a major version.
	release := described.ReplaceAllString(kite, "")
	running, err := parseVersion(release)
	if err != nil || !strings.Contains(release, ".") {
		return nil
	}
	if !want.admits(running) {
		return fmt.Errorf("%s %s requires Kite %s, and this is Kite %s; "+
			"install a Kite in that range, or a version of the %s made for this one",
			kind, name, requires, kite, kind)
	}
	return nil
}

// described matches what git describe adds to a tag for a build made after
// it: the commits since, the commit, and whether the tree was dirty. Such a
// build carries at least the tagged release, so the tag is what is compared,
// rather than a pre-release that the suffix would otherwise read as.
var described = regexp.MustCompile(`(-\d+-g[0-9a-f]+)?(-dirty)?$`)

// versionRange is a set of alternatives, any one of which will do; each is a
// list of comparisons that must all hold, as in ">=1.0.0 <2.0.0 || >=3".
type versionRange [][]comparison

type comparison struct {
	op      string
	version version
}

func parseRange(s string) (versionRange, error) {
	var out versionRange
	for alt := range strings.SplitSeq(s, "||") {
		var all []comparison
		for field := range strings.FieldsSeq(strings.ReplaceAll(alt, ",", " ")) {
			op := "="
			for _, candidate := range []string{">=", "<=", ">", "<", "="} {
				if rest, ok := strings.CutPrefix(field, candidate); ok {
					op, field = candidate, rest
					break
				}
			}
			v, err := parseVersion(field)
			if err != nil {
				return nil, err
			}
			all = append(all, comparison{op: op, version: v})
		}
		if len(all) == 0 {
			return nil, fmt.Errorf("an alternative is empty")
		}
		out = append(out, all)
	}
	return out, nil
}

func (r versionRange) admits(v version) bool {
	for _, all := range r {
		ok := true
		for _, c := range all {
			n := v.compare(c.version)
			switch c.op {
			case ">=":
				ok = ok && n >= 0
			case ">":
				ok = ok && n > 0
			case "<=":
				ok = ok && n <= 0
			case "<":
				ok = ok && n < 0
			default:
				ok = ok && n == 0
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// version is a semantic version; a missing minor or patch counts as zero.
type version struct {
	core [3]int
	pre  []string
}

func parseVersion(s string) (version, error) {
	var v version
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	s, _, _ = strings.Cut(s, "+") // build metadata does not order
	s, pre, hasPre := strings.Cut(s, "-")
	parts := strings.Split(s, ".")
	if s == "" || len(parts) > 3 {
		return v, fmt.Errorf("%q is not a version", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, fmt.Errorf("%q is not a version", s)
		}
		v.core[i] = n
	}
	if hasPre {
		if pre == "" {
			return v, fmt.Errorf("%q has an empty pre-release", s)
		}
		v.pre = strings.Split(pre, ".")
	}
	return v, nil
}

// compare orders versions the way semver does: a pre-release comes before
// the release it leads up to.
func (v version) compare(o version) int {
	for i := range v.core {
		if n := cmp.Compare(v.core[i], o.core[i]); n != 0 {
			return n
		}
	}
	switch {
	case len(v.pre) == 0 && len(o.pre) == 0:
		return 0
	case len(v.pre) == 0:
		return 1
	case len(o.pre) == 0:
		return -1
	}
	for i := range min(len(v.pre), len(o.pre)) {
		a, aErr := strconv.Atoi(v.pre[i])
		b, bErr := strconv.Atoi(o.pre[i])
		var n int
		switch {
		case aErr == nil && bErr == nil:
			n = cmp.Compare(a, b)
		case aErr == nil:
			n = -1 // numeric identifiers come first
		case bErr == nil:
			n = 1
		default:
			n = cmp.Compare(v.pre[i], o.pre[i])
		}
		if n != 0 {
			return n
		}
	}
	return cmp.Compare(len(v.pre), len(o.pre))
}
