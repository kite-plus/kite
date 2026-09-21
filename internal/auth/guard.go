package auth

import (
	"fmt"
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
	account *Account
	csrf    *http.CrossOriginProtection
	now     func() time.Time

	mu       sync.Mutex
	failures int
	blocked  time.Time

	// verifying serializes password checks. Argon2id is memory-hard on
	// purpose, so a server that ran one per request would be handing anyone
	// who can open connections a way to exhaust its memory.
	verifying sync.Mutex
}

// Attempts are free until there have been this many in a row, after which
// each one costs the next a doubling delay. Five is well past a typo and far
// short of anything a person does deliberately.
const freeAttempts = 5

const maxBackoff = 5 * time.Minute

// TooManyAttempts reports a sign-in refused because of earlier failures. It
// carries how long is left, which the client shows rather than guessing.
type TooManyAttempts struct{ RetryAfter time.Duration }

func (e *TooManyAttempts) Error() string {
	return fmt.Sprintf("auth: too many attempts; try again in %s", e.RetryAfter.Round(time.Second))
}

// New returns a guard for an account, which may be nil for an open server.
func New(account *Account) *Guard { return NewWithClock(account, time.Now) }

// NewWithClock is [New] with the clock supplied, for tests that need to reach
// an expiry without waiting for one.
func NewWithClock(account *Account, now func() time.Time) *Guard {
	return &Guard{account: account, csrf: http.NewCrossOriginProtection(), now: now}
}

// Required reports whether requests have to be signed in.
func (g *Guard) Required() bool { return g != nil && g.account != nil }

// User is the name that can sign in, or "" when nobody can.
func (g *Guard) User() string {
	if !g.Required() {
		return ""
	}
	return g.account.user
}

// Source says where the account came from, for a server that has to tell an
// operator which one it is using.
func (g *Guard) Source() string {
	if !g.Required() {
		return ""
	}
	return g.account.source
}

// Session reads and checks the session a request carries.
func (g *Guard) Session(r *http.Request) (Session, error) {
	if !g.Required() {
		return Session{}, ErrNoSession
	}
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return Session{}, ErrNoSession
	}
	return g.account.Parse(c.Value, g.now())
}

// SignIn checks credentials and, when they are right, sets the cookie.
func (g *Guard) SignIn(w http.ResponseWriter, r *http.Request, user, password string, remember bool) (Session, error) {
	if !g.Required() {
		return Session{}, ErrNoAccount
	}

	now := g.now()
	if wait := g.wait(now); wait > 0 {
		return Session{}, &TooManyAttempts{RetryAfter: wait}
	}

	g.verifying.Lock()
	err := g.account.Verify(user, password)
	g.verifying.Unlock()
	if err != nil {
		g.failed(now)
		return Session{}, err
	}

	lifetime := SessionLifetime
	if remember {
		lifetime = RememberLifetime
	}
	token, expires, err := g.account.Issue(now, lifetime)
	if err != nil {
		return Session{}, err
	}

	g.succeeded()
	http.SetCookie(w, cookie(token, expires, remember, secureRequest(r)))
	return Session{User: g.account.user, Expires: expires}, nil
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

// wait reports how long a refused attempt has left.
func (g *Guard) wait(now time.Time) time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	if now.Before(g.blocked) {
		return g.blocked.Sub(now)
	}
	return 0
}

func (g *Guard) failed(now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.failures++
	if over := g.failures - freeAttempts; over > 0 {
		delay := time.Second << min(over-1, 10)
		g.blocked = now.Add(min(delay, maxBackoff))
	}
}

func (g *Guard) succeeded() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.failures, g.blocked = 0, time.Time{}
}

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
