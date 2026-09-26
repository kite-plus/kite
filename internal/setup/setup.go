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

// errWrongKind reports a flow asked to finish in the way it was not started
// for: an account for a site that does not exist yet, or the reverse.
var errWrongKind = errors.New("setup: this flow does not finish that way")

// Flow is the first run of one server.
//
// It is created only when there is something to guide: a server with an
// account, or one on localhost where an open studio is nobody else's
// business, has no flow and no gate. The exception is a folder with no site
// in it yet, where the first run is describing the site itself.
type Flow struct {
	root  string
	guard *auth.Guard

	// create writes a new project, for a flow that makes the site rather
	// than an account; defaults are what its form starts from.
	create   func(Site) error
	defaults Site
	created  chan struct{}

	mu   sync.Mutex
	done bool
}

// Site is what a new site is created with.
type Site struct {
	Title       string
	Description string
	BaseURL     string
	Language    string
}

// NewSite starts a flow for a folder that holds no site yet.
//
// Finishing it describes the site and asks for no account: the studio of a
// folder being started from scratch is on this machine only, and create
// writes the project the answers describe.
func NewSite(defaults Site, create func(Site) error) *Flow {
	return &Flow{create: create, defaults: defaults, created: make(chan struct{})}
}

// CreatesSite reports whether finishing this flow creates the site rather
// than an account.
func (f *Flow) CreatesSite() bool { return f != nil && f.create != nil }

// Defaults are what the form of a new site starts from.
func (f *Flow) Defaults() Site { return f.defaults }

// CreateSite writes the new project and closes the flow.
func (f *Flow) CreateSite(site Site) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case f.done:
		return ErrDone
	case f.create == nil:
		return errWrongKind
	}
	if err := f.create(site); err != nil {
		return err
	}
	f.done = true
	close(f.created)
	return nil
}

// Created is closed once the site exists, which is what the command that
// started the flow waits for before it serves the site.
func (f *Flow) Created() <-chan struct{} { return f.created }

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
	switch {
	case f.done:
		return nil, ErrDone
	case f.create != nil:
		return nil, errWrongKind
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
