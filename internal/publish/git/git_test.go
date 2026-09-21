package git_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/publish"
	gitpub "github.com/kite-plus/kite/internal/publish/git"
)

// run executes git in a directory and fails the test if it does not succeed.
func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepo makes a repository with one committed post, which is the state a
// publish normally starts from.
func newRepo(t *testing.T) string {
	t.Helper()
	if !gitpub.Available() {
		t.Skip("git is not installed")
	}

	root := t.TempDir()
	run(t, root, "init", "-q", "-b", "main", root)
	run(t, root, "config", "user.name", "Test")
	run(t, root, "config", "user.email", "test@example.com")
	run(t, root, "config", "commit.gpgsign", "false")

	write(t, root, "kite.yaml", "site:\n  title: Test\n  baseURL: https://example.com\n")
	write(t, root, "content/posts/first/index.md", "---\ntitle: First\n---\n\nbody\n")
	write(t, root, ".gitignore", "/.kite/\n/public/\n")
	run(t, root, "add", "-A")
	run(t, root, "commit", "-q", "-m", "initial")
	return root
}

func newPublisher(root string) *gitpub.Publisher {
	return gitpub.New(gitpub.Options{
		Root: root,
		Now:  func() time.Time { return time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC) },
	})
}

// plan is Preflight, failing the test on an unexpected error.
func plan(t *testing.T, p *gitpub.Publisher, req publish.Request) *publish.Plan {
	t.Helper()
	out, err := p.Preflight(t.Context(), req)
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	return out
}

func codes(problems []publish.Problem) []string {
	out := make([]string, 0, len(problems))
	for _, p := range problems {
		out = append(out, p.Code)
	}
	return out
}

// This is the promise that decides whether anyone points Kite at a repository
// they care about: a publish commits what it was asked to commit, and touches
// nothing else. An author with work in progress must be able to publish one
// post without it swallowing their afternoon.
func TestPublishCommitsOnlyWhatItWasAskedTo(t *testing.T) {
	root := newRepo(t)
	pub := newPublisher(root)

	// What Kite changed.
	write(t, root, "content/posts/second/index.md", "---\ntitle: Second\n---\n\nnew\n")

	// What the author is in the middle of, in three different states.
	write(t, root, "content/posts/first/index.md", "---\ntitle: First\n---\n\nedited by hand\n")
	write(t, root, "notes.txt", "a scratch file\n")
	write(t, root, "static/logo.svg", "<svg/>\n")
	run(t, root, "add", "static/logo.svg")

	p := plan(t, pub, publish.Request{
		Paths:   []string{"content/posts/second"},
		Message: "publish: second",
	})
	if !p.OK() {
		t.Fatalf("plan is not usable: %v", codes(p.Problems))
	}

	if _, err := pub.Apply(t.Context(), p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// The commit carries exactly one file.
	committed := strings.Split(run(t, root, "show", "--name-only", "--pretty=format:", "HEAD"), "\n")
	committed = slices.DeleteFunc(committed, func(s string) bool { return strings.TrimSpace(s) == "" })
	if want := []string{"content/posts/second/index.md"}; !slices.Equal(committed, want) {
		t.Errorf("the commit carries %v, want %v", committed, want)
	}

	// The hand edit is still uncommitted, and still says what it said.
	edited, err := os.ReadFile(filepath.Join(root, "content/posts/first/index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(edited), "edited by hand") {
		t.Error("the author's own edit was lost")
	}
	if !strings.Contains(run(t, root, "status", "--porcelain"), "content/posts/first/index.md") {
		t.Error("the author's own edit was committed")
	}

	// What they staged is still staged, and nothing else.
	staged := run(t, root, "diff", "--name-only", "--cached")
	if staged != "static/logo.svg" {
		t.Errorf("the staging area now holds %q, want only static/logo.svg", staged)
	}

	// And their scratch file is still untracked.
	if !strings.Contains(run(t, root, "status", "--porcelain"), "?? notes.txt") {
		t.Error("an untracked file was swept into the commit")
	}
}

// A repository in a state Kite does not understand is refused, with every
// reason reported at once rather than one per attempt.
func TestARepositoryThatIsNotReadyIsRefused(t *testing.T) {
	t.Run("not a repository", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "content/posts/a/index.md", "x")

		p := plan(t, newPublisher(root), publish.Request{Paths: []string{"content/posts/a"}})
		if !slices.Contains(codes(p.Problems), publish.CodeNotARepository) {
			t.Errorf("problems = %v, want not_a_repository", codes(p.Problems))
		}
		if p.OK() {
			t.Error("the plan says it can be applied")
		}
	})

	t.Run("detached head", func(t *testing.T) {
		root := newRepo(t)
		run(t, root, "checkout", "-q", "--detach")
		write(t, root, "content/posts/a/index.md", "x")

		p := plan(t, newPublisher(root), publish.Request{Paths: []string{"content/posts/a"}})
		if !slices.Contains(codes(p.Problems), publish.CodeDetachedHead) {
			t.Errorf("problems = %v, want detached_head", codes(p.Problems))
		}
	})

	t.Run("merge in progress", func(t *testing.T) {
		root := newRepo(t)
		// A merge left half done is exactly the state a commit must not join.
		gitDir := filepath.Join(root, ".git")
		if err := os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte("deadbeef\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		write(t, root, "content/posts/a/index.md", "x")

		p := plan(t, newPublisher(root), publish.Request{Paths: []string{"content/posts/a"}})
		if !slices.Contains(codes(p.Problems), publish.CodeOperationInFlight) {
			t.Errorf("problems = %v, want operation_in_flight", codes(p.Problems))
		}
	})

	t.Run("nothing changed", func(t *testing.T) {
		root := newRepo(t)
		p := plan(t, newPublisher(root), publish.Request{Paths: []string{"content/posts/first"}})
		if !slices.Contains(codes(p.Problems), publish.CodeNothingToPublish) {
			t.Errorf("problems = %v, want nothing_to_publish", codes(p.Problems))
		}
	})
}

// A refused plan must not have done anything on the way to being refused.
func TestARefusedPlanLeavesTheRepositoryExactlyAsItWas(t *testing.T) {
	root := newRepo(t)
	run(t, root, "checkout", "-q", "--detach")
	before := run(t, root, "rev-parse", "HEAD")

	write(t, root, "content/posts/a/index.md", "x")
	pub := newPublisher(root)

	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/a"}})
	if _, err := pub.Apply(t.Context(), p); err == nil {
		t.Fatal("Apply accepted a plan it should have refused")
	}

	if after := run(t, root, "rev-parse", "HEAD"); after != before {
		t.Errorf("HEAD moved from %s to %s", before, after)
	}
	if staged := run(t, root, "diff", "--name-only", "--cached"); staged != "" {
		t.Errorf("something was staged: %q", staged)
	}
	if _, err := os.Stat(filepath.Join(root, ".kite", "publish.lock")); err == nil {
		t.Error("the lock was left behind")
	}
}

// Staging your own version of a file Kite is about to commit is worth being
// told about: the commit will carry what is on disk, under Kite's message.
func TestStagingTheSameFileIsWarnedAboutRatherThanRefused(t *testing.T) {
	root := newRepo(t)
	write(t, root, "content/posts/first/index.md", "---\ntitle: First\n---\n\nmine\n")
	run(t, root, "add", "content/posts/first/index.md")

	p := plan(t, newPublisher(root), publish.Request{Paths: []string{"content/posts/first"}})
	if !slices.Contains(codes(p.Warnings), publish.CodeStagedElsewhere) {
		t.Errorf("warnings = %v, want staged_elsewhere", codes(p.Warnings))
	}
	// A warning is not a refusal: the author may well mean it.
	if !p.OK() {
		t.Errorf("a warning blocked the publish: %v", codes(p.Problems))
	}
}

// Two publishes at once must not interleave.
func TestASecondPublishWaitsRatherThanInterleaving(t *testing.T) {
	root := newRepo(t)
	write(t, root, "content/posts/second/index.md", "new\n")

	if err := os.MkdirAll(filepath.Join(root, ".kite"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".kite", "publish.lock"), []byte("pid 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/second"}})

	_, err := pub.Apply(t.Context(), p)
	if err == nil {
		t.Fatal("a second publish ran while another holds the lock")
	}
	var problem publish.Problem
	if !asProblem(err, &problem) || problem.Code != publish.CodeLocked {
		t.Errorf("err = %v, want a locked problem", err)
	}
	// The lock is not broken on the holder's behalf.
	if _, statErr := os.Stat(filepath.Join(root, ".kite", "publish.lock")); statErr != nil {
		t.Error("the lock was removed by the process that did not take it")
	}
}

func asProblem(err error, out *publish.Problem) bool {
	var p publish.Problem
	if errors.As(err, &p) {
		*out = p
		return true
	}
	return false
}

// A publish that a hook refuses has to leave nothing behind. An author who
// declines a publish should be able to forget it happened.
func TestARefusedCommitRestoresWhatWasThereBefore(t *testing.T) {
	root := newRepo(t)
	write(t, root, "content/posts/second/index.md", "new\n")

	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"),
		[]byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	before := run(t, root, "status", "--porcelain")
	head := run(t, root, "rev-parse", "HEAD")

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/second"}})
	if _, err := pub.Apply(t.Context(), p); err == nil {
		t.Fatal("the hook's refusal was not reported")
	}

	if after := run(t, root, "status", "--porcelain"); after != before {
		t.Errorf("the working tree changed\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if now := run(t, root, "rev-parse", "HEAD"); now != head {
		t.Error("a commit was made despite the hook refusing")
	}
	if _, err := os.Stat(filepath.Join(root, ".kite", "publish.lock")); err == nil {
		t.Error("the lock was left behind")
	}
}

// Removing a post has to reach the published site, so a deletion is a change
// like any other.
func TestDeletingAPostIsPublishedAsADeletion(t *testing.T) {
	root := newRepo(t)
	if err := os.RemoveAll(filepath.Join(root, "content", "posts", "first")); err != nil {
		t.Fatal(err)
	}

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/first"}})
	if !p.OK() {
		t.Fatalf("a deletion was not publishable: %v", codes(p.Problems))
	}
	if _, err := pub.Apply(t.Context(), p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if out := run(t, root, "show", "--name-status", "--pretty=format:", "HEAD"); !strings.HasPrefix(out, "D\t") {
		t.Errorf("the commit records %q, want a deletion", out)
	}
}

// Pushing to a remote that cannot be reached must fail quickly. A server
// process asked for a password has nobody to ask, and a request that waits
// forever for one is the worst outcome this package can produce.
func TestAnUnreachableRemoteFailsQuicklyRatherThanHanging(t *testing.T) {
	root := newRepo(t)
	// Port 1 refuses immediately, so this measures the code path and not the
	// network.
	run(t, root, "remote", "add", "origin", "https://127.0.0.1:1/nope.git")
	write(t, root, "content/posts/second/index.md", "new\n")

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/second"}, Push: true})
	if !p.OK() {
		t.Fatalf("plan: %v", codes(p.Problems))
	}

	start := time.Now()
	result, err := pub.Apply(t.Context(), p)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("pushing to an unreachable remote reported success")
	}
	if elapsed > 20*time.Second {
		t.Errorf("the push took %s, which is indistinguishable from a hang", elapsed)
	}

	// The commit is the half that succeeded, and saying so is what stops an
	// author repeating a commit they already have.
	if result == nil || result.Commit == "" {
		t.Fatal("the result does not report the commit that was made")
	}
	if result.Pushed {
		t.Error("the result claims a push that did not happen")
	}
	if out := run(t, root, "show", "--name-only", "--pretty=format:", "HEAD"); !strings.Contains(out, "second") {
		t.Error("the commit was not made")
	}
}

// A remote that has moved on is reported before anything is attempted.
func TestARemoteWithNewCommitsIsReportedBeforePublishing(t *testing.T) {
	origin := t.TempDir()
	run(t, origin, "init", "-q", "--bare", "-b", "main", origin)

	root := newRepo(t)
	run(t, root, "remote", "add", "origin", origin)
	run(t, root, "push", "-q", "--set-upstream", "origin", "main")

	// Somebody else pushes. A second clone is how that actually happens.
	other := t.TempDir()
	run(t, other, "clone", "-q", origin, other)
	run(t, other, "config", "user.name", "Other")
	run(t, other, "config", "user.email", "other@example.com")
	write(t, other, "content/posts/theirs/index.md", "theirs\n")
	run(t, other, "add", "-A")
	run(t, other, "commit", "-q", "-m", "theirs")
	run(t, other, "push", "-q")

	// This repository now fetches, so it can see it is behind.
	run(t, root, "fetch", "-q")
	write(t, root, "content/posts/second/index.md", "new\n")

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{Paths: []string{"content/posts/second"}, Push: true})

	if !slices.Contains(codes(p.Warnings), publish.CodeRemoteMoved) {
		t.Errorf("warnings = %v, want remote_moved", codes(p.Warnings))
	}

	// The push is refused rather than forced, and the commit survives.
	result, err := pub.Apply(t.Context(), p)
	if err == nil {
		t.Fatal("the push was not refused")
	}
	var problem publish.Problem
	if !asProblem(err, &problem) || problem.Code != publish.CodeRemoteMoved {
		t.Errorf("err = %v, want remote_moved", err)
	}
	if result == nil || result.Commit == "" {
		t.Error("the commit that was made is not reported")
	}

	// Nothing on the remote was overwritten.
	if out := run(t, origin, "log", "--oneline", "-1", "main"); !strings.Contains(out, "theirs") {
		t.Errorf("the remote's tip is %q, want the other author's commit", out)
	}
}

// The happy path, end to end, against a real remote.
func TestAPublishReachesTheRemote(t *testing.T) {
	origin := t.TempDir()
	run(t, origin, "init", "-q", "--bare", "-b", "main", origin)

	root := newRepo(t)
	run(t, root, "remote", "add", "origin", origin)
	write(t, root, "content/posts/second/index.md", "---\ntitle: Second\n---\n\nnew\n")

	pub := newPublisher(root)
	p := plan(t, pub, publish.Request{
		Paths:   []string{"content/posts/second"},
		Message: "publish: second",
		Push:    true,
	})
	if !p.OK() {
		t.Fatalf("plan: %v", codes(p.Problems))
	}

	result, err := pub.Apply(t.Context(), p)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.Pushed {
		t.Error("the result does not report the push")
	}

	// The file is on the remote.
	if out := run(t, origin, "show", "main:content/posts/second/index.md"); !strings.Contains(out, "Second") {
		t.Errorf("the remote does not have the post: %q", out)
	}

	state, err := pub.State(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.Committed != publish.StepDone {
		t.Errorf("committed = %q, want done", state.Committed)
	}
	if state.Pushed != publish.StepDone {
		t.Errorf("pushed = %q, want done", state.Pushed)
	}
	if state.Ahead != 0 {
		t.Errorf("ahead = %d, want 0 after a push", state.Ahead)
	}
}

// A file the host will refuse is reported before the commit, not after the
// push that carries it is rejected.
func TestAFileTooLargeToPushIsReportedBeforeCommitting(t *testing.T) {
	root := newRepo(t)

	// Sparse, so the test costs a few bytes rather than 101MB.
	big := filepath.Join(root, "static", "huge.bin")
	if err := os.MkdirAll(filepath.Dir(big), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(101 << 20); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	p := plan(t, newPublisher(root), publish.Request{Paths: []string{"static/huge.bin"}})
	if !slices.Contains(codes(p.Problems), publish.CodeQuotaExceeded) {
		t.Fatalf("problems = %v, want quota_exceeded", codes(p.Problems))
	}
	if p.OK() {
		t.Error("the plan says it can be applied")
	}
	if fix := p.Problems[0].Fix; fix == "" {
		t.Error("the problem says what is wrong but not what to do about it")
	}
}
