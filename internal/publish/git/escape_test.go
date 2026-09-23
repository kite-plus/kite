package git_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/publish"
)

// refusingHook installs a pre-commit hook that says why it refuses, and
// leaves a mark whenever it runs.
func refusingHook(t *testing.T, root string) (ran string) {
	t.Helper()
	ran = filepath.Join(t.TempDir(), "hook-ran")
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ntouch '" + ran + "'\necho 'lint: 3 problems in src/app.ts' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return ran
}

// A refusal has to say it came from a hook, and what the hook said, or the
// author is left guessing why a publish failed.
func TestAHookRefusalSaysSo(t *testing.T) {
	root := newRepo(t)
	refusingHook(t, root)
	write(t, root, "content/posts/second/index.md", "new\n")

	pub := newPublisher(root)
	_, err := pub.Apply(t.Context(), plan(t, pub, publish.Request{Paths: []string{"content/posts/second"}}))
	var problem publish.Problem
	if !asProblem(err, &problem) || problem.Code != publish.CodeHookRefused {
		t.Fatalf("err = %v, want hook_refused", err)
	}
	for _, want := range []string{"pre-commit", "lint: 3 problems"} {
		if !strings.Contains(problem.Detail, want) {
			t.Errorf("the refusal does not mention %q: %s", want, problem.Detail)
		}
	}
}

// Publishing without the hooks commits exactly what was asked, leaves every
// other change where it was, and never runs the hook it was asked to skip.
func TestSkippingHooksPublishesOnlyWhatWasAsked(t *testing.T) {
	root := newRepo(t)
	ran := refusingHook(t, root)
	head := run(t, root, "rev-parse", "HEAD")

	write(t, root, "content/posts/second/index.md", "---\ntitle: Second\n---\n\nnew\n")
	write(t, root, "content/posts/first/index.md", "---\ntitle: First\n---\n\nedited by hand\n")
	write(t, root, "notes.txt", "a scratch file\n")
	write(t, root, "static/logo.svg", "<svg/>\n")
	run(t, root, "add", "static/logo.svg")
	before := strings.Split(run(t, root, "status", "--porcelain"), "\n")

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/second"}, Message: "publish: second", SkipHooks: true})
	if !p.SkipHooks {
		t.Fatal("the plan does not say the hooks will be skipped")
	}
	result, err := pub.Apply(t.Context(), p)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the hook ran although it was to be skipped")
	}

	if parent := run(t, root, "rev-parse", "HEAD^"); parent != head || result.Commit != run(t, root, "rev-parse", "HEAD") {
		t.Errorf("the commit is not a single step on the branch: parent %s, want %s", parent, head)
	}
	if subject := run(t, root, "log", "-1", "--format=%s"); subject != "publish: second" {
		t.Errorf("subject = %q", subject)
	}
	committed := run(t, root, "show", "--name-only", "--pretty=format:", "HEAD")
	if committed != "content/posts/second/index.md" {
		t.Errorf("the commit carries %q", committed)
	}

	// Everything else is exactly as the author left it, and the published
	// file is clean rather than looking staged back to its old self.
	after := strings.Split(run(t, root, "status", "--porcelain"), "\n")
	want := slices.DeleteFunc(before, func(line string) bool { return strings.Contains(line, "content/posts/second") })
	if !slices.Equal(after, want) {
		t.Errorf("status after the publish:\n%s\nwant:\n%s", strings.Join(after, "\n"), strings.Join(want, "\n"))
	}
}

func TestSkippingHooksPublishesADeletion(t *testing.T) {
	root := newRepo(t)
	refusingHook(t, root)
	if err := os.RemoveAll(filepath.Join(root, "content/posts/first")); err != nil {
		t.Fatal(err)
	}

	pub := newPublisher(root)
	if _, err := pub.Apply(t.Context(), plan(t, pub, publish.Request{
		Paths: []string{"content/posts/first"}, SkipHooks: true,
	})); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if files := run(t, root, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(files, "content/posts/first") {
		t.Errorf("the deleted post is still committed:\n%s", files)
	}
	if status := run(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("the tree is not clean:\n%s", status)
	}
}

// A repository with no commit yet gets its first one.
func TestSkippingHooksCanMakeTheFirstCommit(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q", "-b", "main", root)
	run(t, root, "config", "user.name", "Test")
	run(t, root, "config", "user.email", "test@example.com")
	run(t, root, "config", "commit.gpgsign", "false")
	write(t, root, "content/posts/first/index.md", "first\n")

	pub := newPublisher(root)
	if _, err := pub.Apply(t.Context(), plan(t, pub, publish.Request{
		Paths: []string{"content/posts/first"}, Message: "publish: first", SkipHooks: true,
	})); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if log := run(t, root, "log", "--format=%s"); log != "publish: first" {
		t.Errorf("history = %q", log)
	}
}
