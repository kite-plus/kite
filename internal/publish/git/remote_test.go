package git_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/publish"
	gitpub "github.com/kite-plus/kite/internal/publish/git"
)

// shared is a repository that has pushed to a remote, and a second clone of
// that remote for somebody else to push from.
type shared struct {
	root, origin, other string
	pub                 *gitpub.Publisher
}

func newShared(t *testing.T) *shared {
	t.Helper()
	origin := t.TempDir()
	run(t, origin, "init", "-q", "--bare", "-b", "main", origin)

	root := newRepo(t)
	run(t, root, "remote", "add", "origin", origin)
	run(t, root, "push", "-q", "--set-upstream", "origin", "main")

	other := t.TempDir()
	run(t, other, "clone", "-q", origin, other)
	run(t, other, "config", "commit.gpgsign", "false")

	return &shared{root: root, origin: origin, other: other, pub: newPublisher(root)}
}

// theyPush commits a change in the other clone and pushes it.
func (s *shared) theyPush(t *testing.T, rel, body, message string) {
	t.Helper()
	write(t, s.other, rel, body)
	run(t, s.other, "add", "-A")
	// run sets one identity for every command, so theirs is given here.
	run(t, s.other, "commit", "-q", "--author", "Other <other@example.com>", "-m", message)
	run(t, s.other, "push", "-q")
}

// publishes commits paths and pushes them, returning what Apply said.
func (s *shared) publishes(t *testing.T, message string, paths ...string) (*publish.Result, error) {
	t.Helper()
	p := plan(t, s.pub, publish.Request{Paths: paths, Message: message, Push: true, Force: true})
	if len(p.Problems) > 0 {
		t.Fatalf("plan: %v", codes(p.Problems))
	}
	return s.pub.Apply(t.Context(), p)
}

func codeOf(err error) string {
	var p publish.Problem
	if asProblem(err, &p) {
		return p.Code
	}
	return ""
}

// The case the design promises a single click for: the remote moved on, but
// nothing it changed is anything this publish changed.
func TestARemoteThatMovedOnElsewhereIsOfferedARebase(t *testing.T) {
	s := newShared(t)
	s.theyPush(t, "content/posts/theirs/index.md", "theirs\n", "add their post")

	// What the author is in the middle of, in every state git has.
	write(t, s.root, "content/posts/first/index.md", "---\ntitle: First\n---\n\nedited by hand\n")
	write(t, s.root, "notes.txt", "a scratch file\n")
	write(t, s.root, "static/logo.svg", "<svg/>\n")
	run(t, s.root, "add", "static/logo.svg")
	before := run(t, s.root, "status", "--porcelain")

	write(t, s.root, "content/posts/second/index.md", "---\ntitle: Second\n---\n\nnew\n")
	result, err := s.publishes(t, "publish: second", "content/posts/second")
	if codeOf(err) != publish.CodeRemoteMoved {
		t.Fatalf("err = %v, want remote_moved", err)
	}
	change := result.Remote
	if change == nil {
		t.Fatal("the refused push does not say what the remote has")
	}
	if change.Behind != 1 || len(change.Commits) != 1 || change.Commits[0].Subject != "add their post" {
		t.Errorf("remote = %+v, want their one commit", change)
	}
	if change.Commits[0].Author != "Other" || change.Upstream != "origin/main" {
		t.Errorf("commit = %+v on %s", change.Commits[0], change.Upstream)
	}
	if !change.Rebase || change.Blocked != nil || len(change.Overlap) != 0 {
		t.Fatalf("a rebase is not offered: rebase %v, blocked %+v, overlap %v",
			change.Rebase, change.Blocked, change.Overlap)
	}
	made := result.Commit

	pushed, err := s.pub.Push(t.Context(), publish.PushRequest{Rebase: true})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if !pushed.Pushed || !pushed.Rebased || pushed.Commit == made {
		t.Errorf("result = %+v, want a new commit, rebased and pushed", pushed)
	}

	// The remote has their commit and then this one, and nothing was forced.
	log := run(t, s.origin, "log", "--format=%s|%an", "main")
	if want := "publish: second|Test\nadd their post|Other\ninitial|Test"; log != want {
		t.Errorf("remote history:\n%s\nwant:\n%s", log, want)
	}
	if run(t, s.root, "rev-parse", "HEAD") != run(t, s.origin, "rev-parse", "main") {
		t.Error("the branch and the remote disagree after the push")
	}

	// Their post arrived, and the author's work in progress is exactly as it
	// was: the same edit, the same scratch file, the same thing staged.
	if _, err := os.Stat(filepath.Join(s.root, "content/posts/theirs/index.md")); err != nil {
		t.Error("the remote's new file is not in the working tree")
	}
	if after := run(t, s.root, "status", "--porcelain"); after != before {
		t.Errorf("the working tree changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	edited, _ := os.ReadFile(filepath.Join(s.root, "content/posts/first/index.md"))
	if !strings.Contains(string(edited), "edited by hand") {
		t.Error("the author's own edit was lost")
	}
}

// Both sides changed the same file. Which version is right is not something
// a program should decide, so the remote's side is shown and nothing moves.
func TestAnOverlapIsShownAndNothingIsReplayed(t *testing.T) {
	s := newShared(t)
	s.theyPush(t, "content/posts/first/index.md", "---\ntitle: First\n---\n\ntheir wording\n", "reword first")

	write(t, s.root, "content/posts/first/index.md", "---\ntitle: First\n---\n\nmy wording\n")
	result, err := s.publishes(t, "publish: first", "content/posts/first")
	if codeOf(err) != publish.CodeRemoteMoved {
		t.Fatalf("err = %v, want remote_moved", err)
	}
	change := result.Remote
	if change.Rebase || change.Blocked == nil || change.Blocked.Code != publish.CodeRemoteOverlap {
		t.Fatalf("remote = %+v, want an overlap that blocks the rebase", change)
	}
	if !slices.Equal(change.Overlap, []string{"content/posts/first/index.md"}) {
		t.Errorf("overlap = %v", change.Overlap)
	}
	if !strings.Contains(change.Diff, "+their wording") {
		t.Errorf("the diff does not show the remote's side:\n%s", change.Diff)
	}

	head := run(t, s.root, "rev-parse", "HEAD")
	if _, err := s.pub.Push(t.Context(), publish.PushRequest{Rebase: true}); codeOf(err) != publish.CodeRemoteOverlap {
		t.Errorf("err = %v, want remote_overlap", err)
	}
	if run(t, s.root, "rev-parse", "HEAD") != head {
		t.Error("the branch moved although nothing could be replayed")
	}
	if out := run(t, s.origin, "log", "-1", "--format=%s", "main"); out != "reword first" {
		t.Errorf("the remote's tip is %q", out)
	}
}

// Only a single publish commit is replayed. Commits the author made
// themselves are theirs to rebase.
func TestOnlyASinglePublishIsReplayed(t *testing.T) {
	s := newShared(t)
	s.theyPush(t, "content/posts/theirs/index.md", "theirs\n", "add their post")

	write(t, s.root, "static/robots.txt", "User-agent: *\n")
	run(t, s.root, "add", "static/robots.txt")
	run(t, s.root, "commit", "-q", "-m", "their own commit")

	write(t, s.root, "content/posts/second/index.md", "new\n")
	result, err := s.publishes(t, "publish: second", "content/posts/second")
	if codeOf(err) != publish.CodeRemoteMoved {
		t.Fatalf("err = %v, want remote_moved", err)
	}
	if change := result.Remote; change.Rebase || change.Blocked == nil || change.Blocked.Code != publish.CodeUnpushedCommits {
		t.Errorf("remote = %+v, want unpushed_commits", change)
	}
}

// Bringing the remote's files into the working tree would overwrite
// whatever the author has not committed in them.
func TestUncommittedWorkInAFileTheRemoteChangedBlocksTheRebase(t *testing.T) {
	s := newShared(t)
	s.theyPush(t, "kite.yaml", "site:\n  title: Theirs\n  baseURL: https://example.com\n", "retitle")

	mine := "site:\n  title: Mine, not saved anywhere else\n  baseURL: https://example.com\n"
	write(t, s.root, "kite.yaml", mine)

	write(t, s.root, "content/posts/second/index.md", "new\n")
	result, err := s.publishes(t, "publish: second", "content/posts/second")
	if codeOf(err) != publish.CodeRemoteMoved {
		t.Fatalf("err = %v, want remote_moved", err)
	}
	change := result.Remote
	if change.Rebase || change.Blocked == nil || change.Blocked.Code != publish.CodeLocalChanges {
		t.Fatalf("remote = %+v, want local_changes", change)
	}
	if !strings.Contains(change.Blocked.Detail, "kite.yaml") {
		t.Errorf("the reason does not name the file: %q", change.Blocked.Detail)
	}

	if _, err := s.pub.Push(t.Context(), publish.PushRequest{Rebase: true}); codeOf(err) != publish.CodeLocalChanges {
		t.Errorf("err = %v, want local_changes", err)
	}
	if data, _ := os.ReadFile(filepath.Join(s.root, "kite.yaml")); string(data) != mine {
		t.Errorf("the author's edit is gone:\n%s", data)
	}
}

// A replay rebuilds the commit's tree by hand, and a deletion is the part of
// a change that is easiest to drop on the way.
func TestAReplayedDeletionStaysDeleted(t *testing.T) {
	s := newShared(t)
	s.theyPush(t, "content/posts/theirs/index.md", "theirs\n", "add their post")

	if err := os.RemoveAll(filepath.Join(s.root, "content/posts/first")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.publishes(t, "publish: remove first", "content/posts/first"); codeOf(err) != publish.CodeRemoteMoved {
		t.Fatalf("err = %v, want remote_moved", err)
	}
	if _, err := s.pub.Push(t.Context(), publish.PushRequest{Rebase: true}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	files := run(t, s.origin, "ls-tree", "-r", "--name-only", "main")
	if strings.Contains(files, "content/posts/first/") {
		t.Errorf("the deleted post is back on the remote:\n%s", files)
	}
	if !strings.Contains(files, "content/posts/theirs/index.md") {
		t.Errorf("their post is missing from the remote:\n%s", files)
	}
	if status := run(t, s.root, "status", "--porcelain"); status != "" {
		t.Errorf("the working tree is not clean after the replay:\n%s", status)
	}
}

// A commit whose push never happened can be pushed on its own, which
// publishing the same paths again cannot do: they are committed already.
func TestACommitThatWasNotPushedCanBePushedLater(t *testing.T) {
	s := newShared(t)
	write(t, s.root, "content/posts/second/index.md", "new\n")
	p := plan(t, s.pub, publish.Request{Paths: []string{"content/posts/second"}, Message: "publish: second"})
	if _, err := s.pub.Apply(t.Context(), p); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	result, err := s.pub.Push(t.Context(), publish.PushRequest{})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if !result.Pushed || result.Rebased {
		t.Errorf("result = %+v, want pushed as it was", result)
	}
	if out := run(t, s.origin, "log", "-1", "--format=%s", "main"); out != "publish: second" {
		t.Errorf("the remote's tip is %q", out)
	}

	if _, err := s.pub.Push(t.Context(), publish.PushRequest{}); codeOf(err) != publish.CodeNothingToPush {
		t.Errorf("err = %v, want nothing_to_push", err)
	}
}

// A GitHub repository that deploys to Pages reports whether a push is live,
// and the delivery state says so once GitHub has.
func TestAPushToGitHubPagesIsReportedDeployedWhenItIs(t *testing.T) {
	root := newRepo(t)
	run(t, root, "remote", "add", "origin", "https://github.com/acme/site.git")
	// Pushed, as far as this repository can tell, without reaching GitHub.
	run(t, root, "update-ref", "refs/remotes/origin/main", "HEAD")
	run(t, root, "config", "branch.main.remote", "origin")
	run(t, root, "config", "branch.main.merge", "refs/heads/main")
	head := run(t, root, "rev-parse", "HEAD")

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/site/deployments":
			fmt.Fprintf(w, `[{"id": 7, "sha": %q}]`, head)
		case "/repos/acme/site/deployments/7/statuses":
			fmt.Fprint(w, `[{"id": 1, "state": "success", "environment_url": "https://acme.github.io/site/"}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()

	pub := gitpub.New(gitpub.Options{Root: root, GitHubAPI: api.URL})
	deadline := time.Now().Add(5 * time.Second)
	for {
		state, err := pub.State(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if state.Deployed == publish.StepDone {
			if state.DeployedURL != "https://acme.github.io/site/" {
				t.Errorf("deployed url = %q", state.DeployedURL)
			}
			break
		}
		if state.Deployed != publish.StepPending || time.Now().After(deadline) {
			t.Fatalf("deployed = %q, want done", state.Deployed)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A host that does not report deployments is not asked, and the step says it
// does not apply rather than waiting forever.
func TestAnyOtherHostLeavesDeploymentNotApplicable(t *testing.T) {
	s := newShared(t)
	var asked atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { asked.Add(1) }))
	defer api.Close()

	state, err := gitpub.New(gitpub.Options{Root: s.root, GitHubAPI: api.URL}).State(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.Pushed != publish.StepDone || state.Deployed != publish.StepNotApplicable {
		t.Errorf("pushed %q, deployed %q; want done and not_applicable", state.Pushed, state.Deployed)
	}
	time.Sleep(50 * time.Millisecond)
	if n := asked.Load(); n != 0 {
		t.Errorf("GitHub was asked %d times about a repository that is not there", n)
	}
}
