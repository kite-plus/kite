// Package setup carries a server through its first run.
//
// A project made of files is installed by running a command in its directory,
// and `kite init` asks the questions there. A server is not: a container has
// no terminal to type into, and the person who brought it up is looking at a
// browser. This is the same installation, asked for through the API instead.
//
// It exists because of what the server would otherwise do. An admin with no
// account on an address other than localhost is open to whoever finds the
// port, so kite refuses to start one. That refusal is right and is kept: the
// server starts, but until setup is finished the only thing the API will
// answer is setup itself, and setup needs a token that was printed on the
// operator's own console. Nobody else can reach it, and nobody has to shell
// into a container to get going.
package setup

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kite-plus/kite/internal/auth"
)

// TokenFile is where the token is kept, relative to the project root.
//
// It lives beside the account, under a directory every kite project already
// ignores, so a token cannot reach a repository. It is written as well as
// printed because a console scrolls, and `docker exec ... cat` is then still
// a way to find it.
const TokenFile = ".kite/secrets/setup-token"

// TokenEnv names the token instead of having one generated, for a deployment
// that would rather put the secret in its compose file than read it out of a
// log.
const TokenEnv = "KITE_SETUP_TOKEN"

// ErrBadToken reports a setup attempt with the wrong token.
var ErrBadToken = errors.New("setup: incorrect setup token")

// ErrDone reports setup that has already happened. It is not an error the
// operator can act on, and it is what a second browser tab gets.
var ErrDone = errors.New("setup: this server is already set up")

// Flow is the first run of one server.
//
// It is created only when there is something to guide: a server with an
// account, or one on localhost where an open studio is nobody else's
// business, has no flow and no gate.
type Flow struct {
	root     string
	guard    *auth.Guard
	now      func() time.Time
	attempts *auth.Throttle

	mu    sync.Mutex
	token string
	done  bool
}

// New starts a flow for a project whose studio has no account yet.
func New(root string, guard *auth.Guard) (*Flow, error) {
	return NewWithClock(root, guard, time.Now)
}

// NewWithClock is [New] with the clock supplied, for tests that need to reach
// a backoff without waiting for one.
func NewWithClock(root string, guard *auth.Guard, now func() time.Time) (*Flow, error) {
	if guard.Required() {
		return nil, auth.ErrAlreadyConfigured
	}
	token, err := token(root)
	if err != nil {
		return nil, err
	}
	return &Flow{
		root:     root,
		guard:    guard,
		now:      now,
		attempts: auth.NewThrottle(5),
		token:    token,
	}, nil
}

// Pending reports whether this server is still waiting to be set up. A nil
// flow is a server that was never waiting, which is the ordinary case.
func (f *Flow) Pending() bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.done
}

// Token is the secret that has to be presented to finish setup.
func (f *Flow) Token() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.token
}

// Check reports whether a token is the one this flow is waiting for.
//
// Wrong tokens are counted, so that a token nobody was given cannot be
// guessed at machine speed over a network.
func (f *Flow) Check(presented string) error {
	if !f.Pending() {
		return ErrDone
	}
	now := f.now()
	if err := f.attempts.Check(now); err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(presented), []byte(f.Token())) != 1 {
		f.attempts.Refused(now)
		return ErrBadToken
	}
	f.attempts.Accepted()
	return nil
}

// Complete creates the account, installs it into the running guard and closes
// the flow.
//
// The lock is held across the whole of it. Two browsers posting the form at
// the same moment would otherwise both write an account, and the password
// that ends up stored would be whichever one happened to finish second.
func (f *Flow) Complete(user, password string) (*auth.Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.done {
		return nil, ErrDone
	}

	// The account is written before anything else is changed, so that a
	// server that dies partway comes back up configured rather than back at
	// setup with the password already taken.
	account, err := auth.SetPassword(f.root, user, password)
	if err != nil {
		return nil, err
	}
	if err := f.guard.Adopt(account); err != nil {
		return nil, err
	}
	f.done, f.token = true, ""

	// The token is spent. Leaving it behind would leave a secret on disk that
	// no longer opens anything, which is the kind of thing that gets copied
	// into a backup and worried about later.
	if err := os.Remove(filepath.Join(f.root, filepath.FromSlash(TokenFile))); err != nil &&
		!errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return account, nil
}

// token returns the token this run should use.
//
// An existing one is kept: a restarted container that minted a new token
// every time would invalidate the link its operator had just copied, and the
// file is readable only by the account the server runs as.
func token(root string) (string, error) {
	if fromEnv := strings.TrimSpace(os.Getenv(TokenEnv)); fromEnv != "" {
		return fromEnv, nil
	}

	path := filepath.Join(root, filepath.FromSlash(TokenFile))
	switch stored, err := os.ReadFile(path); {
	case err == nil:
		if existing := strings.TrimSpace(string(stored)); existing != "" {
			return existing, nil
		}
	case !errors.Is(err, fs.ErrNotExist):
		return "", err
	}

	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// Base32 without padding: 32 characters a person can read off a terminal
	// and type into a form without wondering about case or punctuation.
	fresh := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf))

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(fresh+"\n"), 0o600); err != nil {
		return "", err
	}
	return fresh, nil
}
