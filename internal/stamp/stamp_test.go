package stamp_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/stamp"
)

const (
	commitA = "0123456789abcdef0123456789abcdef01234567"
	commitB = "89abcdef0123456789abcdef0123456789abcdef"
)

func env(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

// A host names the commit it builds, which outranks whatever a checkout of
// it says.
func TestTheCommitAHostNamesComesFirst(t *testing.T) {
	root := repo(t)
	for name, vars := range map[string]map[string]string{
		"GitHub Actions":   {"GITHUB_SHA": commitA},
		"Vercel":           {"VERCEL_GIT_COMMIT_SHA": commitA},
		"Cloudflare Pages": {"CF_PAGES_COMMIT_SHA": commitA},
		"Netlify":          {"COMMIT_REF": commitA},
		"GitLab CI":        {"CI_COMMIT_SHA": commitA},
		"upper case":       {"GITHUB_SHA": strings.ToUpper(commitA)},
		"two, in order":    {"GITHUB_SHA": commitA, "COMMIT_REF": commitB},
	} {
		if got := stamp.Commit(root, env(vars)); got != commitA {
			t.Errorf("%s: commit = %q, want %q", name, got, commitA)
		}
	}
}

func TestAValueThatIsNotACommitIsPassedOver(t *testing.T) {
	got := stamp.Commit(t.TempDir(), env(map[string]string{"GITHUB_SHA": "main", "COMMIT_REF": commitB}))
	if got != commitB {
		t.Errorf("commit = %q, want the next variable's %q", got, commitB)
	}
}

func TestARepositoryNamesItsHead(t *testing.T) {
	root := repo(t)
	head := strings.TrimSpace(git(t, root, "rev-parse", "HEAD"))
	if got := stamp.Commit(root, env(nil)); got != head {
		t.Errorf("commit = %q, want HEAD %q", got, head)
	}
}

func TestASiteOutsideGitNamesNoCommit(t *testing.T) {
	if got := stamp.Commit(t.TempDir(), env(nil)); got != "" {
		t.Errorf("commit = %q, want none", got)
	}
}

func TestAStampReadsBackAsWritten(t *testing.T) {
	s, err := stamp.Decode(stamp.Encode(commitA))
	if err != nil || s.Commit != commitA {
		t.Errorf("decoded %+v, %v", s, err)
	}
}

// A host may answer an address it has no file for with a page, and an older
// site may have a file of the same name that is something else.
func TestWhatIsNotAStampIsRefused(t *testing.T) {
	for _, body := range []string{
		"",
		"<!doctype html><title>Not found</title>",
		`{"commit": "main"}`,
		`{"commit": ""}`,
		`{"version": "0.1.9"}`,
	} {
		if s, err := stamp.Decode([]byte(body)); err == nil {
			t.Errorf("%q decoded as %+v", body, s)
		}
	}
}

func repo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "-c", "user.name=Kite", "-c", "user.email=kite@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	return root
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}
