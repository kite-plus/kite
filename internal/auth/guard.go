package auth

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Guard is what a server asks whether a request may proceed.
//
// A guard with no account is open, which is what a local preview of a project
// that has never had a password is. It still checks where requests come from:
// a page on the internet can post to a server on localhost, and "it is only
// listening on 127.0.0.1" has never been protection against that.
type Guard struct {
	csrf *http.CrossOriginProtection
	now  func() time.Time

	// account is read under the lock rather than captured, because the
	// first-run setup flow installs one into a server that is already
	// listening. Everywhere else it is written once, at startup.
	mu      sync.RWMutex
	account *Account

	attempts *throttle

	// verifying serializes password checks. Argon2id is memory-hard on
	// purpose, so a server that ran one per request would be handing anyone
	// who can open connections a way to exhaust its memory.
	verifying sync.Mutex
}

// New returns a guard for an account, which may be nil for an open server.
func New(account *Account) *Guard { return NewWithClock(account, time.Now) }

// NewWithClock is [New] with the clock supplied, for tests that need to reach
// an expiry without waiting for one.
func NewWithClock(account *Account, now func() time.Time) *Guard {
	return &Guard{
		csrf:     http.NewCrossOriginProtection(),
		now:      now,
		account:  account,
		attempts: newThrottle(freeAttempts),
	}
}

// current is the account this guard is using, or nil for an open server.
func (g *Guard) current() *Account {
	if g == nil {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.account
}

// Adopt installs an account into a running guard, which is how the studio
// stops being open the moment first-run setup finishes rather than at the
// next restart.
//
// It refuses to replace an existing account: changing a password is a
// deliberate act with a command of its own, and letting it happen here would
// turn setup into a way to take a configured server over.
func (g *Guard) Adopt(account *Account) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.account != nil {
		return ErrAlreadyConfigured
	}
	g.account = account
	return nil
}

// Required reports whether requests have to be signed in.
func (g *Guard) Required() bool { return g.current() != nil }

// User is the name that can sign in, or "" when nobody can.
func (g *Guard) User() string {
	a := g.current()
	if a == nil {
		return ""
	}
	return a.user
}

// Source says where the account came from, for a server that has to tell an
// operator which one it is using.
func (g *Guard) Source() string {
	a := g.current()
	if a == nil {
		return ""
	}
	return a.source
}

// Session reads and checks the session a request carries.
func (g *Guard) Session(r *http.Request) (Session, error) {
	a := g.current()
	if a == nil {
		return Session{}, ErrNoSession
	}
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return Session{}, ErrNoSession
	}
	return a.Parse(c.Value, g.now())
}

// SignIn checks credentials and, when they are right, sets the cookie.
func (g *Guard) SignIn(w http.ResponseWriter, r *http.Request, user, password string, remember bool) (Session, error) {
	account := g.current()
	if account == nil {
		return Session{}, ErrNoAccount
	}

	now := g.now()
	if err := g.attempts.Check(now); err != nil {
		return Session{}, err
	}

	g.verifying.Lock()
	err := account.Verify(user, password)
	g.verifying.Unlock()
	if err != nil {
		g.attempts.Refused(now)
		return Session{}, err
	}

	lifetime := SessionLifetime
	if remember {
		lifetime = RememberLifetime
	}
	token, expires, err := account.Issue(now, lifetime)
	if err != nil {
		return Session{}, err
	}

	g.attempts.Accepted()
	http.SetCookie(w, cookie(token, expires, remember, secureRequest(r)))
	return Session{User: account.user, Expires: expires}, nil
}

// Start issues a session without checking a password, for a caller that has
// just proved itself some other way -- today, the first-run setup flow, which
// created the account it is signing in to a moment earlier.
func (g *Guard) Start(w http.ResponseWriter, r *http.Request) (Session, error) {
	account := g.current()
	if account == nil {
		return Session{}, ErrNoAccount
	}
	token, expires, err := account.Issue(g.now(), SessionLifetime)
	if err != nil {
		return Session{}, err
	}
	http.SetCookie(w, cookie(token, expires, false, secureRequest(r)))
	return Session{User: account.user, Expires: expires}, nil
}

// SignOut clears the cookie. There is nothing else to do: a session is a
// signature rather than a row, so the browser forgetting it is the whole of
// signing out, and the token expires on its own regardless.
func (g *Guard) SignOut(w http.ResponseWriter, r *http.Request) {
	c := cookie("", time.Time{}, false, secureRequest(r))
	c.MaxAge = -1
	c.Expires = time.Unix(1, 0)
	http.SetCookie(w, c)
}

// CheckOrigin rejects a state-changing request that a browser made from
// somewhere else.
//
// The check is the standard library's: Sec-Fetch-Site when the browser sends
// it, the Origin header against the Host otherwise, and requests with neither
// -- curl, a deploy script -- are left alone. That is why a session cookie is
// enough on its own, with no token to plumb through every form.
func (g *Guard) CheckOrigin(r *http.Request) error { return g.csrf.Check(r) }

// Loopback reports whether an address is reachable only from this machine.
//
// An unprotected admin on any other address is reachable by whoever finds the
// port, so this is what decides whether a server may start at all.
func Loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	switch host {
	case "":
		// A bare port means every interface.
		return false
	case "localhost":
		return true
	}
	// A name that is not an address could resolve anywhere, so it counts as
	// public: refusing to start is recoverable, and starting an open admin on
	// a public address is not.
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
