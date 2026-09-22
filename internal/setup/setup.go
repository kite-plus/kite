// Package setup carries a server through its first run.
//
// A project made of files is installed by running a command in its directory,
// and `kite init` asks the questions there. A server is not: a container has
// no terminal to type into, and the person who brought it up is looking at a
// browser. This is the same installation, asked for through the API instead.
//
// What it is not is an authenticated flow. Until somebody finishes it there
// is no account, so there is nobody a request could be checked against, and
// whoever reaches the form first becomes the owner. What the flow does do is
// answer nothing else in the meantime: an unconfigured server hands out no
// content, no drafts and no settings, so the window is a window to claim an
// empty studio rather than to read one.
//
// That window is the cost of being able to install a server from a browser at
// all. It is closed by giving the server an account before it is reachable --
// KITE_ADMIN_USER with KITE_ADMIN_PASSWORD, or `kite auth set-password` --
// which skips this flow entirely.
package setup

import (
	"errors"
	"sync"

	"github.com/kite-plus/kite/internal/auth"
)

// ErrDone reports setup that has already happened. It is not an error the
// operator can act on, and it is what a second browser tab gets.
var ErrDone = errors.New("setup: this server is already set up")

// Flow is the first run of one server.
//
// It is created only when there is something to guide: a server with an
// account, or one on localhost where an open studio is nobody else's
// business, has no flow and no gate.
type Flow struct {
	root  string
	guard *auth.Guard

	mu   sync.Mutex
	done bool
}

// New starts a flow for a project whose studio has no account yet.
func New(root string, guard *auth.Guard) (*Flow, error) {
	if guard.Required() {
		return nil, auth.ErrAlreadyConfigured
	}
	return &Flow{root: root, guard: guard}, nil
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

	account, err := auth.SetPassword(f.root, user, password)
	if err != nil {
		return nil, err
	}
	if err := f.guard.Adopt(account); err != nil {
		return nil, err
	}
	f.done = true
	return account, nil
}
