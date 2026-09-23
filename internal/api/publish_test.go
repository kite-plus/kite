package api_test

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/publish"
)

// git runs a command in a repository, failing the test if it does not work.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=t@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// newRepoProject is a project that is also a git repository with everything
// committed, which is where a publish normally starts.
func newRepoProject(t *testing.T, posts int) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	root := newProject(t, posts)
	git(t, root, "init", "-q", "-b", "main", root)
	git(t, root, "config", "user.name", "Test")
	git(t, root, "config", "user.email", "t@example.com")
	git(t, root, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(root, ".gitignore"), "/.kite/\n/public/\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "initial")
	return root
}

// Publishing one post must carry that post and nothing else, even when the
// author has other work in progress.
func TestPublishingOnePostCommitsOnlyThatPost(t *testing.T) {
	root := newRepoProject(t, 3)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=2", http.StatusOK)
	target, tag := load(t, h, list.Items[0].ID)

	// Kite's change.
	draft := draftOf(target)
	draft.Body = "Edited in the admin.\n"
	if rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+target.ID, draft,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusOK {
		t.Fatalf("save: %d\n%s", rec.Code, rec.Body.String())
	}

	// The author's own, in two different states.
	other, _ := load(t, h, list.Items[1].ID)
	write(t, filepath.Join(root, filepath.FromSlash(other.Locator), "index.md"),
		"---\nid: "+other.ID+"\ntitle: Theirs\nslug: "+other.Slug+"\nstatus: draft\n---\n\nby hand\n")
	write(t, filepath.Join(root, "scratch.txt"), "notes\n")

	rec := send(t, h, http.MethodPost, api.Prefix+"/publish",
		api.PublishBody{IDs: []string{target.ID}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish: %d\n%s", rec.Code, rec.Body.String())
	}

	result := decode[publish.Result](t, rec)
	if result.Commit == "" {
		t.Error("no commit was reported")
	}

	committed := git(t, root, "show", "--name-only", "--pretty=format:", "HEAD")
	if want := string(target.Locator) + "/index.md"; committed != want {
		t.Errorf("the commit carries %q, want %q", committed, want)
	}
	// The subject names the post, not the directory it lives in.
	if subject := git(t, root, "log", "-1", "--format=%s"); !strings.Contains(subject, target.Title) {
		t.Errorf("subject = %q, want it to name %q", subject, target.Title)
	}

	status := git(t, root, "status", "--porcelain")
	for _, want := range []string{string(other.Locator), "scratch.txt"} {
		if !strings.Contains(status, want) {
			t.Errorf("%s was swept into the publish; status:\n%s", want, status)
		}
	}
}

// A repository that cannot be published to says why, before anything happens.
func TestAPublishThatCannotHappenSaysWhy(t *testing.T) {
	root := newRepoProject(t, 2)
	git(t, root, "checkout", "-q", "--detach")
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)
	draft := draftOf(item)
	draft.Body = "changed\n"
	send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": tag})

	rec := send(t, h, http.MethodPost, api.Prefix+"/publish",
		api.PublishBody{IDs: []string{item.ID}}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409\n%s", rec.Code, rec.Body.String())
	}

	body := decode[api.PublishRefused](t, rec)
	if body.Error.Code != api.CodePublishRefused {
		t.Errorf("code = %q", body.Error.Code)
	}
	// The plan comes with the refusal, so every reason is shown at once.
	if body.Plan == nil || len(body.Plan.Problems) == 0 {
		t.Fatal("the refusal carries no plan to explain it")
	}
	if body.Plan.Problems[0].Code != publish.CodeDetachedHead {
		t.Errorf("problem = %q, want detached_head", body.Plan.Problems[0].Code)
	}
	if body.Plan.Problems[0].Fix == "" {
		t.Error("the problem says what is wrong but not what to do")
	}
}

// Preflight has to change nothing at all, which is what makes it safe to run
// whenever a panel is shown.
func TestPreflightLooksWithoutTouchingAnything(t *testing.T) {
	root := newRepoProject(t, 2)
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)
	draft := draftOf(item)
	draft.Body = "changed\n"
	send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": tag})

	head := git(t, root, "rev-parse", "HEAD")
	before := git(t, root, "status", "--porcelain")

	rec := send(t, h, http.MethodPost, api.Prefix+"/publish/preflight",
		api.PublishBody{IDs: []string{item.ID}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d\n%s", rec.Code, rec.Body.String())
	}

	plan := decode[publish.Plan](t, rec)
	if len(plan.Paths) == 0 {
		t.Error("the plan names nothing to publish")
	}
	if plan.Message == "" {
		t.Error("the plan has no commit message")
	}

	if now := git(t, root, "rev-parse", "HEAD"); now != head {
		t.Error("preflight made a commit")
	}
	if now := git(t, root, "status", "--porcelain"); now != before {
		t.Errorf("preflight changed the working tree\nbefore:\n%s\nafter:\n%s", before, now)
	}
}

// A project that is not a repository reports that rather than failing oddly.
func TestPublishingWithoutARepositoryIsReportedPlainly(t *testing.T) {
	h, _ := newWritableServer(t, newProject(t, 2))

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	rec := send(t, h, http.MethodPost, api.Prefix+"/publish",
		api.PublishBody{IDs: []string{list.Items[0].ID}}, nil)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	body := decode[api.PublishRefused](t, rec)
	if body.Plan == nil || body.Plan.Problems[0].Code != publish.CodeNotARepository {
		t.Errorf("problems = %v, want not_a_repository", body.Plan)
	}
}

// A push refused because the remote moved on says what the remote has, and
// when none of it touches the published post, asking for a rebase pushes the
// post on top of it.
func TestAPushRefusedByAMovedRemoteCanBeReplayedOnIt(t *testing.T) {
	root := newRepoProject(t, 2)
	origin := t.TempDir()
	git(t, origin, "init", "-q", "--bare", "-b", "main", origin)
	git(t, root, "remote", "add", "origin", origin)
	git(t, root, "push", "-q", "--set-upstream", "origin", "main")

	other := t.TempDir()
	git(t, other, "clone", "-q", origin, other)
	git(t, other, "config", "commit.gpgsign", "false")
	write(t, filepath.Join(other, "content", "posts", "theirs", "index.md"),
		"---\nid: 01J8KQ2P3R4S5T6V7W8X9YZ990\ntitle: Theirs\nslug: theirs\nstatus: published\n---\n\nFrom elsewhere.\n")
	git(t, other, "add", "-A")
	git(t, other, "commit", "-q", "-m", "add theirs")
	git(t, other, "push", "-q")

	h, _ := newWritableServer(t, root)
	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)
	draft := draftOf(item)
	draft.Body = "Edited in the admin.\n"
	if rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusOK {
		t.Fatalf("save: %d\n%s", rec.Code, rec.Body.String())
	}

	rec := send(t, h, http.MethodPost, api.Prefix+"/publish",
		api.PublishBody{IDs: []string{item.ID}, Push: true}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("publish: %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	refused := decode[api.PublishRefused](t, rec)
	if refused.Problem == nil || refused.Problem.Code != publish.CodeRemoteMoved {
		t.Fatalf("problem = %+v, want remote_moved", refused.Problem)
	}
	if refused.Done == nil || refused.Done.Commit == "" {
		t.Fatal("the commit that was made is not reported")
	}
	if remote := refused.Done.Remote; remote == nil || !remote.Rebase || remote.Behind != 1 {
		t.Fatalf("remote = %+v, want one commit and a rebase on offer", remote)
	}

	rec = send(t, h, http.MethodPost, api.Prefix+"/publish/push", api.PushBody{Rebase: true}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("push: %d\n%s", rec.Code, rec.Body.String())
	}
	if result := decode[publish.Result](t, rec); !result.Pushed || !result.Rebased {
		t.Errorf("result = %+v, want rebased and pushed", result)
	}
	if subjects := git(t, origin, "log", "--format=%s", "main"); !strings.HasPrefix(subjects,
		"publish: "+item.Title+"\nadd theirs\n") {
		t.Errorf("remote history:\n%s", subjects)
	}

	// Their post came with the rebase, and is there to read straight away.
	get[api.Item](t, h, api.Prefix+"/contents/01J8KQ2P3R4S5T6V7W8X9YZ990", http.StatusOK)

	rec = send(t, h, http.MethodPost, api.Prefix+"/publish/push", api.PushBody{}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second push: %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	if again := decode[api.PublishRefused](t, rec); again.Problem == nil || again.Problem.Code != publish.CodeNothingToPush {
		t.Errorf("problem = %+v, want nothing_to_push", again.Problem)
	}
}

// A hook that refuses is reported as the hook's refusal, and an author who
// has read it can publish without the hooks.
func TestAPublishAHookRefusedCanGoAheadWithoutTheHooks(t *testing.T) {
	root := newRepoProject(t, 2)
	ran := filepath.Join(t.TempDir(), "ran")
	write(t, filepath.Join(root, ".git", "hooks", "pre-commit"),
		"#!/bin/sh\ntouch '"+ran+"'\necho 'no commits on Fridays' >&2\nexit 1\n")
	if err := os.Chmod(filepath.Join(root, ".git", "hooks", "pre-commit"), 0o755); err != nil {
		t.Fatal(err)
	}
	h, _ := newWritableServer(t, root)

	list := get[api.List[api.Summary]](t, h, api.Prefix+"/contents?kind=post&limit=1", http.StatusOK)
	item, tag := load(t, h, list.Items[0].ID)
	draft := draftOf(item)
	draft.Body = "Edited in the admin.\n"
	if rec := send(t, h, http.MethodPut, api.Prefix+"/contents/"+item.ID, draft,
		map[string]string{"If-Match": tag}); rec.Code != http.StatusOK {
		t.Fatalf("save: %d\n%s", rec.Code, rec.Body.String())
	}

	rec := send(t, h, http.MethodPost, api.Prefix+"/publish", api.PublishBody{IDs: []string{item.ID}}, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("publish: %d, want 409\n%s", rec.Code, rec.Body.String())
	}
	refused := decode[api.PublishRefused](t, rec)
	if refused.Problem == nil || refused.Problem.Code != publish.CodeHookRefused ||
		!strings.Contains(refused.Problem.Detail, "no commits on Fridays") {
		t.Fatalf("problem = %+v, want the hook's refusal", refused.Problem)
	}
	if err := os.Remove(ran); err != nil {
		t.Fatalf("the hook did not run the first time: %v", err)
	}

	rec = send(t, h, http.MethodPost, api.Prefix+"/publish",
		api.PublishBody{IDs: []string{item.ID}, SkipHooks: true}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish without hooks: %d\n%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the hook ran although it was to be skipped")
	}
	if committed := git(t, root, "show", "--name-only", "--pretty=format:", "HEAD"); committed != string(item.Locator)+"/index.md" {
		t.Errorf("the commit carries %q", committed)
	}
}
