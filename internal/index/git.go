package index

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// gitTimeout bounds the sentinel commands so a slow or wedged repository can
// never block an interactive command.
const gitTimeout = 5 * time.Second

// pathsChangedByGit returns the content paths git says have changed since the
// index last ran.
//
// A pull, checkout or rebase rewrites many files at once. Comparing the two
// commits tells us exactly which ones, so switching branches costs work
// proportional to the diff rather than to the size of the site. An empty result
// means "nothing extra to force": the stat walk still covers everything.
func (ix *Index) pathsChangedByGit(ctx context.Context) (map[string]struct{}, error) {
	previous, err := ix.meta(metaGitHead)
	if err != nil {
		return nil, err
	}
	current, ok := gitHead(ctx, ix.root)
	if !ok || previous == "" || previous == current {
		return nil, nil
	}

	out, ok := git(ctx, ix.root, "diff", "--name-only", previous, current, "--", "content")
	if !ok {
		// The old commit may be gone after a rebase or a shallow fetch. Not
		// knowing the diff is safe: the stat walk is still authoritative.
		return nil, nil
	}

	changed := make(map[string]struct{})
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line != "" {
			changed[line] = struct{}{}
		}
	}
	return changed, nil
}

func (ix *Index) recordGitHead(ctx context.Context) error {
	head, ok := gitHead(ctx, ix.root)
	if !ok {
		return nil
	}
	return ix.setMeta(metaGitHead, head)
}

func gitHead(ctx context.Context, root string) (string, bool) {
	return git(ctx, root, "rev-parse", "HEAD")
}

// git runs a read-only git command, reporting failure as "not available"
// rather than as an error: the index must work in a directory that is not a
// repository, and on a machine with no git at all.
func git(ctx context.Context, root string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	// Never let git block on a credential or editor prompt inside a server.
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")

	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
