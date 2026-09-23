package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/publish"
)

const (
	// shownCommits caps how many of a remote's new commits are listed; the
	// count of all of them is reported separately.
	shownCommits = 20

	// diffLimit caps the diff a refused push reports, which is for reading
	// before deciding what to keep, not for applying.
	diffLimit = 64 << 10
)

// Push sends what is already committed.
//
// It is for a publish whose commit was made and whose push was not: the
// network was down, a credential was missing, or the remote had moved on.
// Publishing the same paths again would find nothing left to commit.
func (p *Publisher) Push(ctx context.Context, req publish.PushRequest) (*publish.Result, error) {
	if !Available() || !p.git.ok(ctx, "rev-parse", "--git-dir") {
		return nil, publish.Problem{
			Code:   publish.CodeNotARepository,
			Detail: "this project is not a git repository",
		}
	}
	branch := p.currentBranch(ctx)
	if branch == "" {
		return nil, publish.Problem{
			Code:   publish.CodeDetachedHead,
			Detail: "HEAD is not on a branch",
			Fix:    "check out a branch first, or create one here with 'git switch -c'",
		}
	}
	remote := p.remote(ctx)
	if remote == "" {
		return nil, publish.Problem{
			Code:   publish.CodeNoRemote,
			Detail: "no remote to push to",
			Fix:    "add one with 'git remote add origin <url>'",
		}
	}
	if ahead, _, ok := p.divergence(ctx, branch, remote); ok && ahead == 0 {
		return nil, publish.Problem{
			Code:   publish.CodeNothingToPush,
			Detail: "there is nothing to push",
			Fix:    "everything committed on " + branch + " is already on " + remote,
		}
	}

	release, err := p.lock.acquire()
	if err != nil {
		return nil, err
	}
	defer release()

	result := &publish.Result{Commit: p.head(ctx), At: p.opts.Now().UTC()}

	if req.Rebase {
		change, upstream, err := p.remoteChange(ctx, remote, branch)
		if err != nil {
			return result, err
		}
		if change.Behind > 0 {
			if !change.Rebase {
				result.Remote = change
				return result, *change.Blocked
			}
			commit, err := p.replay(ctx, branch, result.Commit, upstream, change.Upstream)
			if err != nil {
				result.Remote = change
				return result, err
			}
			result.Commit, result.Rebased = commit, true
		}
	}

	if err := p.send(ctx, result, remote, branch); err != nil {
		return result, err
	}
	return result, nil
}

// send pushes the branch and, when the remote has moved on, finds out what
// it has so the author can decide what to do about it.
func (p *Publisher) send(ctx context.Context, result *publish.Result, remote, branch string) error {
	err := p.push(ctx, remote, branch, result.Commit)
	if err == nil {
		result.Pushed = true
		return nil
	}
	var refused publish.Problem
	if !asProblem(err, &refused) || refused.Code != publish.CodeRemoteMoved {
		return err
	}

	change, _, ferr := p.remoteChange(ctx, remote, branch)
	if ferr != nil {
		// What the push said is still true, and more use than a failed fetch.
		return err
	}
	if change.Behind == 0 {
		// Refused, but not for being behind: a protected branch or a hook
		// on the remote, which git's own message names.
		return publish.Problem{Code: publish.CodeGitFailed, Detail: refused.Detail}
	}
	result.Remote = change
	refused.Detail = fmt.Sprintf("%s has %d commit(s) this branch does not", change.Upstream, change.Behind)
	return refused
}

// remoteChange fetches the branch and works out what the remote has that
// this branch does not, and whether the one unpushed commit can be replayed
// on top of it. It also returns the remote commit it measured against.
//
// It is only ever asked after a push, or by the author: fetching is a network
// call, and a preflight that waited on one would be a preflight nobody runs.
func (p *Publisher) remoteChange(ctx context.Context, remote, branch string) (*publish.RemoteChange, string, error) {
	upstream, err := p.fetch(ctx, remote, branch)
	if err != nil {
		return nil, "", err
	}
	local := p.head(ctx)
	change := &publish.RemoteChange{Upstream: remote + "/" + branch, Commits: []publish.Commit{}}

	base, err := p.git.read(ctx, "merge-base", local, upstream)
	if err != nil || base == "" {
		if change.Behind, err = p.count(ctx, upstream, "^"+local); err != nil {
			return nil, "", err
		}
		if change.Commits, err = p.commits(ctx, upstream, "^"+local); err != nil {
			return nil, "", err
		}
		change.Blocked = &publish.Problem{
			Code:   publish.CodeUnrelatedHistory,
			Detail: "this branch and " + change.Upstream + " share no history",
			Fix:    "one of them was started over; decide in a terminal which one to keep",
		}
		return change, upstream, nil
	}

	if change.Behind, err = p.count(ctx, base+".."+upstream); err != nil || change.Behind == 0 {
		return change, upstream, err
	}
	if change.Commits, err = p.commits(ctx, base+".."+upstream); err != nil {
		return nil, "", err
	}

	ours, err := p.git.read(ctx, "rev-list", base+".."+local)
	if err != nil {
		return nil, "", err
	}
	theirs, err := p.changedBetween(ctx, base, upstream)
	if err != nil {
		return nil, "", err
	}
	mine, err := p.changedBetween(ctx, base, local)
	if err != nil {
		return nil, "", err
	}
	change.Overlap = slices.DeleteFunc(slices.Clone(mine), func(path string) bool {
		return !slices.Contains(theirs, path)
	})

	switch unpushed := strings.Fields(ours); {
	case len(unpushed) == 0:
		change.Blocked = &publish.Problem{
			Code:   publish.CodeNothingToPush,
			Detail: "this branch has nothing " + change.Upstream + " does not",
			Fix:    "pull to catch up with the remote",
		}
	case len(change.Overlap) > 0:
		change.Diff = p.diffOf(ctx, base, upstream, change.Overlap)
		change.Blocked = &publish.Problem{
			Code:   publish.CodeRemoteOverlap,
			Detail: "the remote also changed " + strings.Join(change.Overlap, ", "),
			Fix: "pull, settle the differences, and publish again. Your commit " +
				short(local) + " is safe on this branch.",
		}
	case len(unpushed) > 1 || p.isMerge(ctx, unpushed[0]):
		change.Blocked = &publish.Problem{
			Code: publish.CodeUnpushedCommits,
			Detail: strconv.Itoa(len(unpushed)) +
				" commits on this branch are not pushed yet, and only a single publish is replayed for you",
			Fix: "rebase or merge in a terminal, then push",
		}
	default:
		dirty, err := p.uncommitted(ctx, theirs)
		if err != nil {
			return nil, "", err
		}
		if len(dirty) > 0 {
			change.Blocked = &publish.Problem{
				Code:   publish.CodeLocalChanges,
				Detail: "files the remote changed have uncommitted changes here: " + strings.Join(dirty, ", "),
				Fix:    "commit those changes or set them aside, then try again",
			}
			break
		}
		change.Rebase = true
	}
	return change, upstream, nil
}

// fetch brings the remote's branch up to date and returns the commit it
// points at.
func (p *Publisher) fetch(ctx context.Context, remote, branch string) (string, error) {
	args := []string{"fetch", "--quiet", remote, "refs/heads/" + branch}
	var stderr bytes.Buffer
	if err := p.git.run(ctx, pushTimeout, nil, &stderr, args...); err != nil {
		return "", p.git.wrap(err, args, &stderr)
	}
	// FETCH_HEAD rather than the remote-tracking branch, which exists only
	// where the remote's fetch refspec is the usual one.
	return p.git.read(ctx, "rev-parse", "--verify", "-q", "FETCH_HEAD^{commit}")
}

func (p *Publisher) count(ctx context.Context, revs ...string) (int, error) {
	out, err := p.git.read(ctx, append([]string{"rev-list", "--count"}, revs...)...)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// commits lists the newest commits of a range as a person would recognize
// them.
func (p *Publisher) commits(ctx context.Context, revs ...string) ([]publish.Commit, error) {
	args := append([]string{
		"log", "-z", "--max-count=" + strconv.Itoa(shownCommits),
		"--format=%H%x1f%s%x1f%an%x1f%aI",
	}, revs...)
	out, err := p.git.readRaw(ctx, args...)
	if err != nil {
		return nil, err
	}
	commits := []publish.Commit{}
	for _, record := range strings.Split(out, "\x00") {
		fields := strings.Split(strings.TrimSpace(record), "\x1f")
		if len(fields) != 4 {
			continue
		}
		at, _ := time.Parse(time.RFC3339, fields[3])
		commits = append(commits, publish.Commit{
			Hash: fields[0], Subject: fields[1], Author: fields[2], At: at.UTC(),
		})
	}
	return commits, nil
}

// changedBetween lists the files that differ between two commits. Renames
// are listed as the two paths they touch, since both are changed.
func (p *Publisher) changedBetween(ctx context.Context, from, to string) ([]string, error) {
	out, err := p.git.readRaw(ctx, "diff", "--name-only", "-z", "--no-renames", from, to)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			paths = append(paths, name)
		}
	}
	return paths, nil
}

// diffOf is the remote's side of the files both sides changed.
func (p *Publisher) diffOf(ctx context.Context, from, to string, paths []string) string {
	out, err := p.git.literal().readRaw(ctx, append([]string{"diff", "--no-color", from, to, "--"}, paths...)...)
	if err != nil {
		return ""
	}
	if len(out) > diffLimit {
		out = out[:diffLimit] + "\n[...]\n"
	}
	return out
}

func (p *Publisher) isMerge(ctx context.Context, commit string) bool {
	parents, err := p.git.read(ctx, "rev-list", "--parents", "-n", "1", commit)
	return err != nil || len(strings.Fields(parents)) > 2
}

// uncommitted narrows paths to those with changes of any kind in the index
// or the working tree, untracked files included.
func (p *Publisher) uncommitted(ctx context.Context, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out, err := p.git.readRaw(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var dirty []string
	records := strings.Split(out, "\x00")
	for i := 0; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 {
			continue
		}
		names := []string{record[3:]}
		// A rename or copy is followed by its source, in a record of its own.
		if (record[0] == 'R' || record[0] == 'C') && i+1 < len(records) {
			i++
			names = append(names, records[i])
		}
		for _, name := range names {
			if slices.Contains(paths, name) && !slices.Contains(dirty, name) {
				dirty = append(dirty, name)
			}
		}
	}
	slices.Sort(dirty)
	return dirty, nil
}

// replay puts the one unpushed commit on top of the remote, as a rebase
// would, and returns the commit that replaces it.
//
// git rebase refuses to run with uncommitted changes, and --autostash lifts
// the author's work in progress out of the tree and puts it back. A publish
// promises to leave all of that alone, and has no need to touch it: the
// remote changed no file the commit changed, so the new tree is the remote's
// with the commit's files laid over it. That tree is built in an index of its
// own, the branch moves only if it still points where it did, and then only
// the files the remote changed are brought up to date, none of which hold
// uncommitted work.
func (p *Publisher) replay(ctx context.Context, branch, commit, upstream, name string) (string, error) {
	gitDir, err := p.git.gitDir(ctx)
	if err != nil {
		return "", err
	}
	parent := commit + "^"
	mine, err := p.changedBetween(ctx, parent, commit)
	if err != nil {
		return "", err
	}
	theirs, err := p.changedBetween(ctx, parent, upstream)
	if err != nil {
		return "", err
	}

	index := filepath.Join(gitDir, "kite-replay.index")
	defer func() {
		_ = os.Remove(index)
		_ = os.Remove(index + ".lock")
	}()
	scratch := p.git.literal().with("GIT_INDEX_FILE=" + index)

	if _, err := scratch.call(ctx, "", "read-tree", upstream); err != nil {
		return "", err
	}
	// With no paths, ls-tree would list the whole commit and lay all of it
	// over the remote's tree.
	if len(mine) > 0 {
		entries, err := scratch.call(ctx, "", append([]string{"ls-tree", "-r", "-z", commit, "--"}, mine...)...)
		if err != nil {
			return "", err
		}
		if _, err := scratch.call(ctx, entries, "update-index", "-z", "--index-info"); err != nil {
			return "", err
		}
		var removed []string
		for _, path := range mine {
			if !strings.Contains(entries, "\t"+path+"\x00") {
				removed = append(removed, path)
			}
		}
		if len(removed) > 0 {
			if _, err := scratch.call(ctx, strings.Join(removed, "\x00"),
				"update-index", "-z", "--force-remove", "--stdin"); err != nil {
				return "", err
			}
		}
	}
	tree, err := scratch.call(ctx, "", "write-tree")
	if err != nil {
		return "", err
	}

	// The commit keeps its author, its date and its message; the committer
	// is whoever is replaying it, as with any rebase.
	who, err := p.git.read(ctx, "log", "-1", "--format=%an%x00%ae%x00%ad", "--date=raw", commit)
	if err != nil {
		return "", err
	}
	author := strings.Split(who, "\x00")
	if len(author) != 3 {
		return "", fmt.Errorf("git: cannot read the author of %s", short(commit))
	}
	message, err := p.git.readRaw(ctx, "log", "-1", "--format=%B", commit)
	if err != nil {
		return "", err
	}
	replayed, err := p.git.with(
		"GIT_AUTHOR_NAME="+author[0], "GIT_AUTHOR_EMAIL="+author[1], "GIT_AUTHOR_DATE="+author[2],
	).call(ctx, message, "commit-tree", strings.TrimSpace(tree), "-p", upstream, "-F", "-")
	if err != nil {
		return "", err
	}
	replayed = strings.TrimSpace(replayed)

	// Compare and swap: a commit made in a terminal since the push was
	// refused makes this fail rather than disappear.
	ref := "refs/heads/" + branch
	if _, err := p.git.call(ctx, "", "update-ref", "-m", "kite: replay onto "+name,
		ref, replayed, commit); err != nil {
		return "", publish.Problem{
			Code:   publish.CodeGitFailed,
			Detail: branch + " moved while its commit was being replayed",
			Fix:    "try again",
		}
	}

	if len(theirs) > 0 {
		err := p.retryIndexLock(ctx, func() error {
			_, err := p.git.literal().call(ctx, strings.Join(theirs, "\x00"),
				"restore", "--source="+replayed, "--staged", "--worktree",
				"--pathspec-from-file=-", "--pathspec-file-nul")
			return err
		})
		if err != nil {
			// Back to where it was, so the branch never points at a tree the
			// files on disk do not match.
			_, _ = p.git.call(ctx, "", "update-ref", ref, commit, replayed)
			return "", err
		}
	}
	return replayed, nil
}
