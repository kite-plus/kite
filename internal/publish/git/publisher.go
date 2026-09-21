package git

import (
	"bytes"
	"context"
	"fmt"
	"path"
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

	// Now is injected so a test can make a commit reproducible.
	Now func() time.Time
}

// Publisher commits and pushes content with the git binary.
type Publisher struct {
	opts Options
	git  runner
	lock *lockFile
}

// New returns a publisher over a project.
func New(opts Options) *Publisher {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Publisher{
		opts: opts,
		git:  runner{root: opts.Root},
		lock: newLock(opts.Root),
	}
}

// Name identifies this publisher in configuration and in a plan.
func (p *Publisher) Name() string { return "git" }

// Preflight examines the repository without touching it.
func (p *Publisher) Preflight(ctx context.Context, req publish.Request) (*publish.Plan, error) {
	plan := &publish.Plan{
		Publisher: p.Name(),
		Message:   p.message(req),
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

	if err := p.commit(ctx, plan); err != nil {
		return nil, err
	}
	commit := p.head(ctx)

	result := &publish.Result{
		Commit:    commit,
		Committed: plan.Paths,
		At:        p.opts.Now().UTC(),
	}
	if !plan.Push {
		return result, nil
	}

	if err := p.push(ctx, plan, commit); err != nil {
		// The commit stands: it is on the branch and nothing is lost. Saying
		// the publish failed outright would invite an author to repeat a
		// commit they already have.
		return result, err
	}
	result.Pushed = true
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
		return p.git.wrap(err, args, &stderr)
	}
	return nil
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
func (p *Publisher) push(ctx context.Context, plan *publish.Plan, commit string) error {
	if plan.Remote == "" || plan.Branch == "" {
		return publish.Problem{
			Code:   publish.CodeNoRemote,
			Detail: "no remote to push to",
			Fix:    "add one with 'git remote add origin <url>'",
		}
	}

	args := []string{"push", plan.Remote, plan.Branch}
	// An upstream is set on the first push so later ones need no arguments,
	// and so the branch reports ahead and behind counts.
	if _, err := p.git.read(ctx, "rev-parse", "--abbrev-ref", "@{upstream}"); err != nil {
		args = []string{"push", "--set-upstream", plan.Remote, plan.Branch}
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
				Fix: "pull and try again. Your commit " + short(commit) +
					" is safe on this branch; nothing was overwritten.",
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
		// Whether a deployment finished is something the hosting platform
		// knows and this publisher does not. Reporting a guess would be
		// worse than reporting nothing.
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
		return state, nil
	}
	state.Ahead, state.Behind = ahead, behind
	if ahead == 0 && len(dirty) == 0 {
		state.Pushed = publish.StepDone
	}
	return state, nil
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
		return "publish: " + path.Base(path.Dir(req.Paths[0]))
	default:
		return fmt.Sprintf("publish: %d files", len(req.Paths))
	}
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
