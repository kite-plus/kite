package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/publish"
)

// Options configures a publisher.
type Options struct {
	Root string

	// Remote and Branch override what the repository would choose.
	Remote string
	Branch string

	// Message is a template for the commit subject. Empty uses a default.
	Message string

	// GitHubAPI is where a repository on GitHub is asked whether a push has
	// been deployed. Empty means GitHub's own API.
	GitHubAPI string

	// Now is injected so a test can make a commit reproducible.
	Now func() time.Time
}

// Publisher commits and pushes content with the git binary.
//
// It remembers what GitHub said about deployments, so it is meant to be
// made once and asked many times.
type Publisher struct {
	opts    Options
	git     runner
	lock    *lockFile
	deploys *deployChecker
}

// New returns a publisher over a project.
func New(opts Options) *Publisher {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Publisher{
		opts:    opts,
		git:     runner{root: opts.Root},
		lock:    newLock(opts.Root),
		deploys: newDeployChecker(opts.GitHubAPI, opts.Now),
	}
}

// Name identifies this publisher in configuration and in a plan.
func (p *Publisher) Name() string { return "git" }

// Preflight examines the repository without touching it.
func (p *Publisher) Preflight(ctx context.Context, req publish.Request) (*publish.Plan, error) {
	plan := &publish.Plan{
		Publisher: p.Name(),
		Message:   p.message(req),
		SkipHooks: req.SkipHooks,
	}
	p.preflight(ctx, req, plan)
	return plan, nil
}

// Apply carries out a plan.
//
// The plan is re-examined first. Between looking and acting, an author may
// have started a merge in another window, and a plan is a description of a
// moment rather than a reservation of it.
func (p *Publisher) Apply(ctx context.Context, plan *publish.Plan) (*publish.Result, error) {
	if plan == nil {
		return nil, publish.Problem{Code: publish.CodeGitFailed, Detail: "no plan"}
	}
	if len(plan.Problems) > 0 {
		return nil, plan.Problems[0]
	}
	if len(plan.Paths) == 0 {
		return nil, publish.Problem{
			Code:   publish.CodeNothingToPublish,
			Detail: "there is nothing to publish",
		}
	}

	// One publish at a time, so two of them cannot interleave a commit.
	release, err := p.lock.acquire()
	if err != nil {
		return nil, err
	}
	defer release()

	commit := p.commit
	if plan.SkipHooks {
		commit = p.commitWithoutHooks
	}
	if err := commit(ctx, plan); err != nil {
		return nil, err
	}
	made := p.head(ctx)

	result := &publish.Result{
		Commit:    made,
		Committed: plan.Paths,
		At:        p.opts.Now().UTC(),
	}
	if !plan.Push {
		return result, nil
	}

	if err := p.send(ctx, result, plan.Remote, plan.Branch); err != nil {
		// The commit stands: it is on the branch and nothing is lost. Saying
		// the publish failed outright would invite an author to repeat a
		// commit they already have.
		return result, err
	}
	return result, nil
}

// commit records exactly the planned paths.
//
// `git commit --only -- <paths>` is the whole trick, and it is git's own
// feature rather than something built here. It commits the named paths and
// nothing else: what the author has staged elsewhere stays staged, every
// other dirty file stays dirty, their clean filters and LFS run, and their
// hooks fire. Building a tree by hand would bypass all of that.
//
// A file git has never seen cannot be named that way, so new paths are first
// registered with `git add -N`. That records the intent to add and nothing
// else: it does not touch an entry the author already staged, which plain
// `git add` would overwrite.
func (p *Publisher) commit(ctx context.Context, plan *publish.Plan) error {
	known, err := p.tracked(ctx, plan.Paths)
	if err != nil {
		return err
	}
	introduced := slices.DeleteFunc(slices.Clone(plan.Paths), func(path string) bool {
		return slices.Contains(known, path)
	})

	if len(introduced) > 0 {
		args := append([]string{"add", "--intent-to-add", "--"}, introduced...)
		var stderr bytes.Buffer
		if err := p.git.run(ctx, timeout, nil, &stderr, args...); err != nil {
			return p.git.wrap(err, args, &stderr)
		}
	}

	args := append([]string{"commit", "--only", "-m", plan.Message, "--"}, plan.Paths...)
	var stderr bytes.Buffer
	err = p.retryIndexLock(ctx, func() error {
		stderr.Reset()
		return p.git.run(ctx, timeout, nil, &stderr, args...)
	})
	if err != nil {
		// A hook can refuse, and then the intent-to-add entries are all that
		// is left of the attempt. Taking them back out leaves the repository
		// exactly as it was found, which is what makes a refused publish
		// something an author can ignore.
		p.forget(ctx, introduced)
		if hooks := p.commitHooks(ctx); len(hooks) > 0 {
			return publish.Problem{
				Code:   publish.CodeHookRefused,
				Detail: "the repository's " + strings.Join(hooks, " or ") + " hook refused: " + firstLine(stderr.String()),
				Fix:    "fix what the hook reports, or publish without running the hooks",
			}
		}
		return p.git.wrap(err, args, &stderr)
	}
	return nil
}

// commitHooks lists the hooks that run on a commit and can refuse it.
func (p *Publisher) commitHooks(ctx context.Context) []string {
	dir, err := p.git.read(ctx, "rev-parse", "--path-format=absolute", "--git-path", "hooks")
	if err != nil {
		return nil
	}
	var active []string
	for _, name := range []string{"pre-commit", "prepare-commit-msg", "commit-msg"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			active = append(active, name)
		}
	}
	return active
}

// tracked returns the subset of paths git already knows about.
func (p *Publisher) tracked(ctx context.Context, paths []string) ([]string, error) {
	out, err := p.git.readRaw(ctx, append([]string{"ls-files", "-z", "--"}, paths...)...)
	if err != nil {
		return nil, err
	}
	var known []string
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			known = append(known, name)
		}
	}
	return known, nil
}

// forget removes intent-to-add entries, restoring paths to untracked.
func (p *Publisher) forget(ctx context.Context, paths []string) {
	if len(paths) == 0 {
		return
	}
	args := append([]string{"rm", "--cached", "--ignore-unmatch", "-q", "--"}, paths...)
	_ = p.git.run(ctx, timeout, nil, nil, args...)
}

// push sends the branch to its remote.
//
// --force-with-lease and --force-if-includes together are the only safe form
// of force, and even that is not used here: a push that would overwrite
// something is refused and handed back to the author. What was rejected is
// still committed locally, so nothing is lost by stopping.
func (p *Publisher) push(ctx context.Context, remote, branch, commit string) error {
	if remote == "" || branch == "" {
		return publish.Problem{
			Code:   publish.CodeNoRemote,
			Detail: "no remote to push to",
			Fix:    "add one with 'git remote add origin <url>'",
		}
	}

	args := []string{"push", remote, branch}
	// An upstream is set on the first push so later ones need no arguments,
	// and so the branch reports ahead and behind counts.
	if _, err := p.git.read(ctx, "rev-parse", "--abbrev-ref", "@{upstream}"); err != nil {
		args = []string{"push", "--set-upstream", remote, branch}
	}

	var stdout, stderr bytes.Buffer
	if err := p.git.run(ctx, pushTimeout, &stdout, &stderr, args...); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if strings.Contains(detail, "non-fast-forward") || strings.Contains(detail, "fetch first") ||
			strings.Contains(detail, "rejected") {
			return publish.Problem{
				Code: publish.CodeRemoteMoved,
				Detail: "the remote has commits this branch does not: " +
					firstLine(detail),
				Fix: "your commit " + short(commit) +
					" is safe on this branch; nothing was overwritten",
			}
		}
		return p.git.wrap(err, args, &stderr)
	}
	return nil
}

// State reports how far the content has traveled.
func (p *Publisher) State(ctx context.Context) (*publish.DeliveryState, error) {
	state := &publish.DeliveryState{
		Local:     publish.StepDone,
		Committed: publish.StepPending,
		Pushed:    publish.StepPending,
		Deployed:  publish.StepPending,
		CheckedAt: p.opts.Now().UTC(),
	}

	if !Available() || !p.git.ok(ctx, "rev-parse", "--git-dir") {
		state.Committed = publish.StepNotApplicable
		state.Pushed = publish.StepNotApplicable
		state.Deployed = publish.StepNotApplicable
		return state, nil
	}

	state.Branch = p.currentBranch(ctx)
	state.Remote = p.remote(ctx)

	dirty, err := p.dirtyContent(ctx)
	if err != nil {
		state.LastError = problemOf(err)
		return state, nil
	}
	state.Dirty = dirty
	if len(dirty) == 0 {
		state.Committed = publish.StepDone
	}

	ahead, behind, ok := p.divergence(ctx, state.Branch, state.Remote)
	if !ok {
		state.Pushed = publish.StepNotApplicable
		state.Deployed = publish.StepNotApplicable
		return state, nil
	}
	state.Ahead, state.Behind = ahead, behind
	if ahead == 0 && len(dirty) == 0 {
		state.Pushed = publish.StepDone
	}
	state.Deployed, state.DeployedURL = p.deployed(ctx, state)
	return state, nil
}

// deployed reports whether what was pushed is live, as far as the host says.
//
// Whether a deployment finished is something the host knows and this
// publisher does not, so only a host that reports it is asked: GitHub, for a
// repository that deploys to Pages. Anywhere else the step does not apply,
// which is the truth, where a guess would be worse than nothing.
func (p *Publisher) deployed(ctx context.Context, state *publish.DeliveryState) (publish.Step, string) {
	remote, err := p.git.read(ctx, "remote", "get-url", state.Remote)
	if err != nil {
		return publish.StepNotApplicable, ""
	}
	repo, ok := githubRepo(remote)
	if !ok {
		return publish.StepNotApplicable, ""
	}
	head := p.head(ctx)
	return p.deploys.look(repo, head, state.Pushed == publish.StepDone, func(sha string) bool {
		return p.git.ok(context.Background(), "merge-base", "--is-ancestor", head, sha)
	})
}

// dirtyContent lists what is uncommitted under the paths Kite writes.
//
// Only those paths: the rest of the working tree is the author's business,
// and reporting it here would make a publish panel look like it intends to
// commit things it never will.
func (p *Publisher) dirtyContent(ctx context.Context) ([]string, error) {
	out, err := p.git.readRaw(ctx, append([]string{
		"status", "--porcelain=v1", "-z", "--untracked-files=all", "--",
	}, watched...)...)
	if err != nil {
		return nil, err
	}

	var dirty []string
	for _, entry := range strings.Split(out, "\x00") {
		if len(entry) < 4 {
			continue
		}
		dirty = append(dirty, entry[3:])
	}
	return dirty, nil
}

// watched is what a publish would ever carry.
var watched = []string{"content", "static", "kite.yaml"}

func (p *Publisher) message(req publish.Request) string {
	if req.Message != "" {
		return req.Message
	}
	if p.opts.Message != "" {
		return p.opts.Message
	}
	switch len(req.Paths) {
	case 0:
		return "publish"
	case 1:
		return "publish: " + itemName(req.Paths[0])
	default:
		return fmt.Sprintf("publish: %d files", len(req.Paths))
	}
}

// itemName is what to call a path in a commit subject.
//
// A bundle and a single file are both named by the item they hold, so
// content/posts/hello and content/posts/hello.md both come out as "hello".
func itemName(p string) string {
	name := path.Base(strings.TrimSuffix(p, "/"))
	if name == "index.md" {
		name = path.Base(path.Dir(p))
	}
	return strings.TrimSuffix(name, path.Ext(name))
}

func (p *Publisher) head(ctx context.Context) string {
	out, err := p.git.read(ctx, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

func short(commit string) string {
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}

func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "To ") {
			return t
		}
	}
	return strings.TrimSpace(s)
}

func problemOf(err error) *publish.Problem {
	var p publish.Problem
	if ok := asProblem(err, &p); ok {
		return &p
	}
	return &publish.Problem{Code: publish.CodeGitFailed, Detail: err.Error()}
}
