package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/kite-plus/kite/internal/publish"
)

// commitWithoutHooks records exactly the planned paths without running the
// repository's commit hooks, for an author who has read why a hook refused
// and chosen to publish anyway.
//
// It is the escape hatch of architecture.md §16.3. The commit is built in an
// index of its own, so nothing the author has staged is involved; the paths
// are added through git itself, so clean filters and LFS still run; and the
// branch moves only if it still points where it did, so a commit made in a
// terminal meanwhile makes this fail rather than disappear. Only then are the
// published paths brought up to date in the real index, which leaves every
// other entry in it as it was.
func (p *Publisher) commitWithoutHooks(ctx context.Context, plan *publish.Plan) error {
	gitDir, err := p.git.gitDir(ctx)
	if err != nil {
		return err
	}
	branch := p.currentBranch(ctx)
	if branch == "" {
		return publish.Problem{Code: publish.CodeDetachedHead, Detail: "HEAD is not on a branch"}
	}
	ref := "refs/heads/" + branch
	// An unborn branch has no commit yet: the first one starts from nothing
	// and may only be made if the branch still does not exist.
	old, _ := p.git.read(ctx, "rev-parse", "--verify", "-q", ref)

	index := filepath.Join(gitDir, "kite-publish.index")
	defer func() {
		_ = os.Remove(index)
		_ = os.Remove(index + ".lock")
	}()
	scratch := p.git.literal().with("GIT_INDEX_FILE=" + index)

	base := []string{"read-tree", "--empty"}
	if old != "" {
		base = []string{"read-tree", old}
	}
	if _, err := scratch.call(ctx, "", base...); err != nil {
		return err
	}
	if _, err := scratch.call(ctx, "", append([]string{"add", "--"}, plan.Paths...)...); err != nil {
		return err
	}
	tree, err := scratch.call(ctx, "", "write-tree")
	if err != nil {
		return err
	}

	commitArgs := []string{"commit-tree", strings.TrimSpace(tree), "-F", "-"}
	if old != "" {
		commitArgs = append(commitArgs, "-p", old)
	}
	made, err := p.git.call(ctx, plan.Message+"\n", commitArgs...)
	if err != nil {
		return err
	}
	made = strings.TrimSpace(made)

	if _, err := p.git.call(ctx, "", "update-ref", "-m", "kite: publish", ref, made, old); err != nil {
		return publish.Problem{
			Code:   publish.CodeGitFailed,
			Detail: branch + " moved while the publish was being committed",
			Fix:    "try again",
		}
	}

	err = p.retryIndexLock(ctx, func() error {
		_, err := p.git.literal().call(ctx, "", append([]string{"update-index", "--add", "--remove", "--"}, plan.Paths...)...)
		return err
	})
	if err != nil {
		// Back to where it was, so the branch never holds a commit the index
		// disagrees with; a branch that had no commit goes back to none.
		undo := []string{"update-ref", ref, old, made}
		if old == "" {
			undo = []string{"update-ref", "-d", ref, made}
		}
		_, _ = p.git.call(ctx, "", undo...)
		return err
	}
	return nil
}
