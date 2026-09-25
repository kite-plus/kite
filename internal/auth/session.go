package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// CookieName carries the session. It is prefixed with __Host- nowhere on
// purpose: that prefix requires Secure, and a local preview over http would
// then be unable to sign in at all.
const CookieName = "kite_session"

// How long a session lasts. A browser that was told to stay signed in keeps
// one for a month; one that was not gets a working day and a cookie that goes
// away when the browser does.
const (
	SessionLifetime  = 12 * time.Hour
	RememberLifetime = 30 * 24 * time.Hour
)

// ErrNoSession reports a request carrying no session at all.
var ErrNoSession = errors.New("auth: not signed in")

// ErrBadSession reports a session that was not issued here, was tampered
// with, or has expired.
var ErrBadSession = errors.New("auth: the session is not valid")

// Session is who a request is from and until when.
type Session struct {
	User    string    `json:"-"`
	Expires time.Time `json:"-"`

	// Remember is whether the browser was asked to keep the session past
	// the window it was opened in.
	Remember bool `json:"-"`
}

// claims is the signed payload, kept short because it travels on every
// request. There is no session id: nothing is stored server side, so there
// would be nothing to look it up in.
type claims struct {
	User    string `json:"u"`
	Expires int64  `json:"x"`
	Nonce   string `json:"n"`

	// Remember travels with the token so that one issued again, after a
	// change of password, goes back into the same kind of cookie.
	Remember bool `json:"r,omitempty"`
}

// Issue mints a token for this account.
func (a *Account) Issue(now time.Time, lifetime time.Duration) (string, time.Time, error) {
	return a.issue(now.Add(lifetime), false)
}

// issue mints a token that runs out at expires.
//
// The nonce makes two tokens issued in the same second differ, so that
// signing in again replaces the cookie rather than reproducing it -- a
// session that is byte-identical to one already captured is a session that
// cannot be told apart from a replay.
func (a *Account) issue(expires time.Time, remember bool) (string, time.Time, error) {
	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", time.Time{}, err
	}
	expires = expires.Truncate(time.Second)

	payload, err := json.Marshal(claims{
		User:     a.user,
		Expires:  expires.Unix(),
		Nonce:    base64.RawURLEncoding.EncodeToString(nonce),
		Remember: remember,
	})
	if err != nil {
		return "", time.Time{}, err
	}

	body := base64.RawURLEncoding.EncodeToString(payload)
	return "k1." + body + "." + sign(a.sessionKey(), body), expires, nil
}

// Parse checks a token's signature and expiry and returns who it is for.
func (a *Account) Parse(token string, now time.Time) (Session, error) {
	rest, ok := strings.CutPrefix(token, "k1.")
	if !ok {
		return Session{}, ErrBadSession
	}
	body, mac, ok := strings.Cut(rest, ".")
	if !ok {
		return Session{}, ErrBadSession
	}

	// The signature is checked before the payload is looked at, so nothing
	// unauthenticated is ever decoded, and compared in constant time, so the
	// comparison does not report how much of a forgery was right.
	if !hmac.Equal([]byte(mac), []byte(sign(a.sessionKey(), body))) {
		return Session{}, ErrBadSession
	}

	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return Session{}, ErrBadSession
	}
	var c claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Session{}, ErrBadSession
	}

	expires := time.Unix(c.Expires, 0)
	if !now.Before(expires) {
		return Session{}, ErrBadSession
	}
	// A token still signed by this account's key but naming someone else can
	// only come from a renamed account, and is no longer a session for it.
	if c.User != a.user {
		return Session{}, ErrBadSession
	}
	return Session{User: c.User, Expires: expires, Remember: c.Remember}, nil
}

func sign(key []byte, body string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// cookie packages a token for a browser.
//
// A remembered session is persistent; one that was not asked to be
// remembered has no Max-Age and so lives as long as the browser window,
// while the token inside it expires on its own schedule either way.
func cookie(value string, now, expires time.Time, remember, secure bool) *http.Cookie {
	c := &http.Cookie{
		Name:  CookieName,
		Value: value,
		// The admin and the API sit on different paths of the same origin, so
		// the cookie has to cover both.
		Path:     "/",
		HttpOnly: true,
		// Lax rather than Strict: the admin is a place people arrive at from
		// a link, and a session that is invisible on the first page load
		// looks exactly like being signed out.
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
	if remember {
		c.Expires = expires
		c.MaxAge = int(expires.Sub(now).Seconds())
	}
	return c
}

// secureRequest reports whether the browser reached us over TLS, including
// through a reverse proxy that terminated it.
//
// Trusting a forwarded header cannot weaken anything here: the worst a forged
// one can do is mark the cookie Secure on a plain connection, which makes the
// browser refuse to store it.
func secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	proto, _, _ := strings.Cut(r.Header.Get("X-Forwarded-Proto"), ",")
	return strings.EqualFold(strings.TrimSpace(proto), "https")
}
