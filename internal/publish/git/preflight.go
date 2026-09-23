package git

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kite-plus/kite/internal/publish"
)

// inFlight names the files git leaves behind while an operation is only half
// done. Committing during one of them would fold Kite's change into whatever
// the user is in the middle of.
var inFlight = map[string]string{
	"MERGE_HEAD":       "a merge",
	"REBASE_HEAD":      "a rebase",
	"CHERRY_PICK_HEAD": "a cherry-pick",
	"REVERT_HEAD":      "a revert",
	"BISECT_LOG":       "a bisect",
}

// preflight examines the repository and collects everything wrong with it.
//
// Every check here answers a question with a real answer, never a guess.
// Where a repository is in a state Kite does not understand, the publish
// stops: guessing wrong here means rewriting somebody's work, and no amount
// of convenience is worth that.
func (p *Publisher) preflight(ctx context.Context, req publish.Request, plan *publish.Plan) {
	stop := func(code, detail, fix string) {
		plan.Problems = append(plan.Problems, publish.Problem{Code: code, Detail: detail, Fix: fix})
	}
	warn := func(code, detail, fix string) {
		plan.Warnings = append(plan.Warnings, publish.Problem{Code: code, Detail: detail, Fix: fix})
	}

	if !Available() {
		stop(publish.CodeGitMissing, "git is not installed",
			"install git, or commit and publish this project yourself")
		return
	}
	if !p.git.ok(ctx, "rev-parse", "--git-dir") {
		stop(publish.CodeNotARepository, "this project is not a git repository",
			"run 'git init' here, or set publish.publisher to none")
		return
	}

	gitDir, err := p.git.gitDir(ctx)
	if err != nil {
		stop(publish.CodeGitFailed, err.Error(), "")
		return
	}

	// content/ inside a submodule means the files Kite writes belong to a
	// different repository than the one being committed here.
	if out, err := p.git.read(ctx, "rev-parse", "--show-superproject-working-tree"); err == nil && out != "" {
		stop(publish.CodeSubmodule,
			"this repository is a submodule of "+out,
			"publish from the outer repository instead")
	}

	// A detached head commits onto nothing a branch points at, so the work
	// is reachable only by hash and is lost on the next checkout.
	if !p.git.ok(ctx, "symbolic-ref", "-q", "HEAD") {
		stop(publish.CodeDetachedHead, "HEAD is not on a branch",
			"check out a branch first, or create one here with 'git switch -c'")
	}

	for marker, what := range inFlight {
		if _, err := os.Stat(filepath.Join(gitDir, marker)); err == nil {
			stop(publish.CodeOperationInFlight, what+" is in progress",
				"finish or abort it before publishing")
			break
		}
	}

	plan.Branch = p.currentBranch(ctx)
	plan.Remote = p.remote(ctx)

	p.checkPaths(ctx, req, plan, stop, warn)
	p.checkLFS(ctx, plan, stop)
	p.checkSizes(plan, stop, warn)
	p.checkRemote(ctx, plan, req, warn)
}

// checkPaths keeps a publish to exactly what it was asked to publish.
func (p *Publisher) checkPaths(
	ctx context.Context,
	req publish.Request,
	plan *publish.Plan,
	stop, warn func(code, detail, fix string),
) {
	var carry []string
	for _, path := range req.Paths {
		clean := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
		if clean == "" || clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(clean) {
			stop(publish.CodeGitFailed, "refusing to publish a path outside the project: "+path, "")
			continue
		}
		if !slices.Contains(carry, clean) {
			carry = append(carry, clean)
		}
	}
	slices.Sort(carry)

	// Paths git sees no change in are dropped rather than refused: asking to
	// publish a post that is already committed should be a quiet success.
	changed, err := p.changedPaths(ctx, carry)
	if err != nil {
		stop(publish.CodeGitFailed, err.Error(), "")
		return
	}
	plan.Paths = changed

	if len(changed) == 0 {
		stop(publish.CodeNothingToPublish, "there is nothing to publish",
			"everything asked for is already committed")
		return
	}

	// The user may have staged their own version of a file Kite is about to
	// commit. Committing it would carry their staged content under Kite's
	// message, which is not what either of them meant.
	staged, err := p.git.read(ctx, append([]string{"diff", "--name-only", "--cached", "--"}, changed...)...)
	if err == nil && staged != "" {
		warn(publish.CodeStagedElsewhere,
			"you have already staged changes to: "+strings.Join(strings.Split(staged, "\n"), ", "),
			"publishing will commit what is on disk, including what you staged")
	}
}

// changedPaths narrows a set of paths to those git reports as changed,
// including files that are not tracked yet.
func (p *Publisher) changedPaths(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	// --porcelain is the one status format git promises not to change.
	args := append([]string{"status", "--porcelain=v1", "-z", "--untracked-files=all", "--"}, paths...)
	out, err := p.git.readRaw(ctx, args...)
	if err != nil {
		return nil, err
	}

	var changed []string
	for _, entry := range strings.Split(out, "\x00") {
		if len(entry) < 4 {
			continue
		}
		// "XY <path>", and for a rename the old name follows in its own
		// record, which -z keeps separate.
		name := entry[3:]
		if name != "" && !slices.Contains(changed, name) {
			changed = append(changed, name)
		}
	}
	slices.Sort(changed)
	return changed, nil
}

// checkLFS refuses to publish a file that should become an LFS pointer when
// git-lfs is not installed.
//
// Without the filter git stores the whole file where a pointer belongs. To
// the repository's owner that is corruption, and it is silent.
func (p *Publisher) checkLFS(ctx context.Context, plan *publish.Plan, stop func(code, detail, fix string)) {
	args := append([]string{"check-attr", "filter", "-z", "--"}, plan.Paths...)
	out, err := p.git.read(ctx, args...)
	if err != nil || !strings.Contains(out, "lfs") {
		return
	}
	if p.git.ok(ctx, "lfs", "version") {
		return
	}
	stop(publish.CodeLFSMissing,
		"some of these files are tracked by git-lfs, which is not installed",
		"install git-lfs and run 'git lfs install', or publish these files yourself")
}

// File sizes the hosting platform cares about.
//
// GitHub refuses a push carrying a file over the hard limit outright and
// warns above the soft one. Finding that out from a rejected push, after the
// commit is already made, is the failure this check exists to move earlier.
const (
	hardFileLimit = 100 << 20
	softFileLimit = 50 << 20
)

// checkSizes reports files large enough for the host to object to.
func (p *Publisher) checkSizes(plan *publish.Plan, stop, warn func(code, detail, fix string)) {
	for _, rel := range plan.Paths {
		info, err := os.Stat(filepath.Join(p.opts.Root, filepath.FromSlash(rel)))
		if err != nil || info.IsDir() {
			continue // a deletion has nothing to weigh
		}
		switch {
		case info.Size() > hardFileLimit:
			stop(publish.CodeQuotaExceeded,
				rel+" is "+humanSize(info.Size())+", over the 100MB a push may carry",
				"track it with git-lfs, or keep it out of the repository")
		case info.Size() > softFileLimit:
			warn(publish.CodeQuotaExceeded,
				rel+" is "+humanSize(info.Size())+", which most hosts will warn about",
				"consider git-lfs for files this size")
		}
	}
}

// humanSize is for a sentence a person reads, not for arithmetic.
func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return strconv.FormatFloat(float64(n)/(1<<30), 'f', 1, 64) + "GB"
	case n >= 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + "MB"
	case n >= 1<<10:
		return strconv.FormatFloat(float64(n)/(1<<10), 'f', 1, 64) + "KB"
	default:
		return strconv.FormatInt(n, 10) + "B"
	}
}

// checkRemote reports what stands between this branch and the remote.
func (p *Publisher) checkRemote(ctx context.Context, plan *publish.Plan, req publish.Request, warn func(code, detail, fix string)) {
	if !req.Push {
		plan.Push = false
		return
	}
	if plan.Remote == "" {
		warn(publish.CodeNoRemote, "this repository has no remote",
			"the commit will be made locally; add a remote to publish it")
		plan.Push = false
		return
	}
	plan.Push = true

	// Commits the remote has that this branch does not. Pushing on top of
	// them is refused by git anyway; saying so now is the difference between
	// a plan and a surprise.
	if _, behind, ok := p.divergence(ctx, plan.Branch, plan.Remote); ok && behind > 0 {
		warn(publish.CodeRemoteMoved,
			strconv.Itoa(behind)+" commit(s) on the remote are not in this branch",
			"the push will be refused; if they change none of these files, the commit "+
				"can then go on top of them, and otherwise pull first")
	}
}

// divergence counts how far the branch and its upstream have drifted apart,
// using only what is already fetched. It does not reach the network: a
// preflight that waited on a fetch would be a preflight nobody runs.
func (p *Publisher) divergence(ctx context.Context, branch, remote string) (ahead, behind int, ok bool) {
	if branch == "" || remote == "" {
		return 0, 0, false
	}
	out, err := p.git.read(ctx, "rev-list", "--left-right", "--count",
		branch+"..."+remote+"/"+branch)
	if err != nil {
		return 0, 0, false // no upstream tracked yet, which is not an error
	}
	fields := strings.Fields(out)
	if len(fields) != 2 {
		return 0, 0, false
	}
	ahead, _ = strconv.Atoi(fields[0])
	behind, _ = strconv.Atoi(fields[1])
	return ahead, behind, true
}

func (p *Publisher) currentBranch(ctx context.Context) string {
	out, err := p.git.read(ctx, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// remote picks the remote to publish to: the branch's own upstream when it
// has one, and otherwise the only remote there is.
func (p *Publisher) remote(ctx context.Context) string {
	if p.opts.Remote != "" {
		return p.opts.Remote
	}
	if out, err := p.git.read(ctx, "rev-parse", "--abbrev-ref", "@{upstream}"); err == nil && out != "" {
		if name, _, found := strings.Cut(out, "/"); found {
			return name
		}
	}
	out, err := p.git.read(ctx, "remote")
	if err != nil || out == "" {
		return ""
	}
	names := strings.Fields(out)
	if slices.Contains(names, "origin") {
		return "origin"
	}
	if len(names) == 1 {
		return names[0]
	}
	// Several remotes and no upstream: which one to publish to is a decision
	// only the author can make.
	return ""
}
