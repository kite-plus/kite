package auth

import (
	"errors"
	"sync"
)

// ErrAccountChanged reports a change asked of an account that was replaced
// in the meantime, by another tab or another change.
var ErrAccountChanged = errors.New("auth: the account changed while this was being done")

// ErrFixedAccount reports a change asked of an account the environment
// supplies. It is changed where it is set, and a change written to the file
// would be outranked by it at the next start anyway.
var ErrFixedAccount = errors.New("auth: this account comes from the environment and is changed there")

// ErrMustStayGuarded reports an attempt to take the password away from a
// server that other machines can reach, which would leave its studio open to
// all of them.
var ErrMustStayGuarded = errors.New("auth: a studio other machines can reach has to keep its password")

// Keeper changes the account of a running server.
//
// While a server runs, the account lives in two places: the file under the
// project and the guard checking requests. A change made to only one of them
// is either invisible until a restart or undone by the next one, so every
// change goes through here and reaches both.
type Keeper struct {
	root  string
	guard *Guard

	// open is whether this server may be left with no account, which it may
	// only where nothing but this machine can reach it.
	open bool

	// mu makes changes one at a time, so that two of them cannot both read
	// the same account and each write their own version of it.
	mu sync.Mutex
}

// NewKeeper returns the keeper of a project's account, for a server
// listening on addr and checking requests with guard.
func NewKeeper(root string, guard *Guard, addr string) *Keeper {
	return &Keeper{root: root, guard: guard, open: Loopback(addr)}
}

// MayOpen reports whether the password may be taken away on this server.
func (k *Keeper) MayOpen() bool { return k.open }

// Create gives an open studio its first account.
func (k *Keeper) Create(user, password string) (*Account, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.guard.Required() {
		return nil, ErrAlreadyConfigured
	}
	account, err := SetPassword(k.root, user, password)
	if err != nil {
		return nil, err
	}
	if err := k.guard.Adopt(account); err != nil {
		return nil, err
	}
	return account, nil
}

// Change gives the stored account a new name, a new password, or both; an
// empty one is left as it is.
//
// current is the account whose password the caller confirmed, so a change
// that raced another one is refused rather than applied on top of it. Every
// session ends, because a session is signed with a key derived from the name
// and the password together.
func (k *Keeper) Change(current *Account, user, password string) (*Account, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if err := k.changeable(current); err != nil {
		return nil, err
	}
	next := &Account{user: current.user, password: current.password, secret: current.secret, source: File}
	if user != "" {
		if err := CheckUser(user); err != nil {
			return nil, err
		}
		next.user = user
	}
	if password != "" {
		if err := CheckPassword(password); err != nil {
			return nil, err
		}
		hash, err := Hash(password)
		if err != nil {
			return nil, err
		}
		next.password = hash
	}
	return next, k.install(current, next)
}

// Rotate gives the stored account a new session secret, which ends every
// session at once without changing how anybody signs in.
func (k *Keeper) Rotate(current *Account) (*Account, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if err := k.changeable(current); err != nil {
		return nil, err
	}
	secret, err := newSecret()
	if err != nil {
		return nil, err
	}
	next := &Account{user: current.user, password: current.password, secret: secret, source: File}
	return next, k.install(current, next)
}

// Remove deletes the stored account and leaves the studio open, which only a
// server nobody else can reach may be.
func (k *Keeper) Remove(current *Account) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.open {
		return ErrMustStayGuarded
	}
	if err := k.changeable(current); err != nil {
		return err
	}
	if err := Remove(k.root); err != nil && !errors.Is(err, ErrNoAccount) {
		return err
	}
	return k.guard.Replace(current, nil)
}

// changeable reports why current cannot be changed here, if it cannot.
func (k *Keeper) changeable(current *Account) error {
	switch {
	case current == nil:
		return ErrNoAccount
	case current.source != File:
		return ErrFixedAccount
	case k.guard.current() != current:
		return ErrAccountChanged
	}
	return nil
}

// install writes next and puts it in front of the server.
//
// The file goes first: a server that failed to write it keeps checking the
// account it had, which is still the one on disk.
func (k *Keeper) install(current, next *Account) error {
	if err := next.save(k.root); err != nil {
		return err
	}
	return k.guard.Replace(current, next)
}

// Profile reads how the person using the studio is shown.
func (k *Keeper) Profile() (Profile, error) { return LoadProfile(k.root) }

// SaveProfile stores a profile and returns it as stored.
func (k *Keeper) SaveProfile(p Profile) (Profile, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return SaveProfile(k.root, p)
}

// Avatar reads the picture shown beside the name, and its media type.
func (k *Keeper) Avatar() ([]byte, string, error) { return LoadAvatar(k.root) }

// SaveAvatar stores a picture and returns its media type.
func (k *Keeper) SaveAvatar(data []byte) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return SaveAvatar(k.root, data)
}

// RemoveAvatar forgets the picture.
func (k *Keeper) RemoveAvatar() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return RemoveAvatar(k.root)
}
