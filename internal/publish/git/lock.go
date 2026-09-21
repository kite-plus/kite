package git

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/kite-plus/kite/internal/publish"
)

// lockPath is the advisory lock that keeps two publishes from interleaving.
const lockPath = ".kite/publish.lock"

// indexLockRetries and indexLockBackoff bound the wait for git's own index
// lock, which the user's terminal holds while their command runs.
const (
	indexLockRetries = 5
	indexLockBackoff = 50 * time.Millisecond
)

type lockFile struct{ path string }

func newLock(root string) *lockFile {
	return &lockFile{path: filepath.Join(root, filepath.FromSlash(lockPath))}
}

// acquire takes the lock, returning the function that releases it.
//
// The file is created exclusively, so whoever creates it holds the lock. A
// stale one is reported rather than removed: a lock nobody can explain is
// usually a process still running, and breaking it is how two publishes end
// up interleaved.
func (l *lockFile) acquire() (func(), error) {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil, publish.Problem{
			Code:   publish.CodeLocked,
			Detail: "another publish is in progress",
			Fix: "wait for it to finish. If nothing is running, remove " +
				lockPath + " yourself.",
		}
	}
	if err != nil {
		return nil, err
	}

	// The owner is recorded so the message above can be acted on.
	_, _ = fmt.Fprintf(f, "pid %d at %s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	_ = f.Close()

	return func() { _ = os.Remove(l.path) }, nil
}

// retryIndexLock runs a git command that takes the index lock, backing off
// when the user's own git command holds it.
//
// The stale lock is never removed. index.lock belongs to git, and deleting
// one that a running command owns corrupts the index it was protecting.
func (p *Publisher) retryIndexLock(ctx context.Context, run func() error) error {
	gitDir, _ := p.git.gitDir(ctx)

	var err error
	for attempt := range indexLockRetries {
		if err = run(); err == nil {
			return nil
		}
		if !p.indexLocked(gitDir) {
			return err
		}
		// Jittered, so two processes retrying do not stay in step.
		wait := indexLockBackoff * time.Duration(1<<attempt)
		time.Sleep(wait + time.Duration(rand.N(int64(wait/2))))
	}
	return err
}

// indexLocked reports whether another git command holds the index.
//
// The check is on the file rather than on git's message, which git does not
// promise to keep stable.
func (p *Publisher) indexLocked(gitDir string) bool {
	if gitDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(gitDir, "index.lock"))
	return err == nil
}

// asProblem recovers a publish.Problem from an error chain.
func asProblem(err error, out *publish.Problem) bool {
	var p publish.Problem
	if errors.As(err, &p) {
		*out = p
		return true
	}
	return false
}
