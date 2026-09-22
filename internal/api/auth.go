package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/kite-plus/kite/internal/auth"
)

// SessionInfo is what a client is told about signing in.
//
// The account name is absent until the caller has one, because a name is half
// of a credential and an anonymous request has no business learning it.
// Required is reported regardless, since a client cannot decide whether to
// show a sign-in form without it.
type SessionInfo struct {
	Required      bool      `json:"required"`
	Authenticated bool      `json:"authenticated"`
	User          string    `json:"user,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitzero"`
}

// Credentials is a sign-in attempt.
type Credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`

	// Remember keeps the session past the browser window, for a person on
	// their own machine who would rather not sign in every morning.
	Remember bool `json:"remember,omitempty"`
}

// publicPaths are the endpoints that answer without a session.
//
// The description of the API is here because it is the same document for
// every server of this version and carries nothing about this one; the rest
// are how a caller gets a session in the first place.
var publicPaths = map[string]bool{
	OpenAPIPath:     true,
	"/setup":        true,
	"/auth/session": true,
	"/auth/login":   true,
	"/auth/logout":  true,
}

// allow decides whether a request may reach a handler at all.
//
// Two checks, in this order: where the request came from, then who it is
// from. Origin first because a cross-site request carrying a valid cookie is
// exactly the case that has to be refused, and it would pass the second.
func (s *Server) allow(w http.ResponseWriter, r *http.Request) bool {
	if err := s.auth.CheckOrigin(r); err != nil {
		fail(w, http.StatusForbidden, CodeCrossOrigin,
			"this request came from another site and was refused")
		return false
	}
	// A server still waiting to be set up has no account, so there is nobody
	// a session could belong to and nothing here may be answered on trust.
	// Only the way through setup responds until it has an owner.
	if s.setup.Pending() && !setupPaths[r.URL.Path] {
		fail(w, http.StatusForbidden, CodeSetupRequired,
			"this server has not been set up yet")
		return false
	}
	if !s.auth.Required() || publicPaths[r.URL.Path] {
		return true
	}
	if _, err := s.auth.Session(r); err != nil {
		fail(w, http.StatusUnauthorized, CodeUnauthorized, "sign in to continue")
		return false
	}
	return true
}

// handleSession reports whether this server needs a sign-in, and whether the
// caller already has one.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	info := SessionInfo{Required: s.auth.Required()}
	if session, err := s.auth.Session(r); err == nil {
		info.Authenticated = true
		info.User = session.User
		info.ExpiresAt = session.Expires
	}
	writeJSON(w, http.StatusOK, info)
}

// handleLogin exchanges credentials for a session cookie.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Required() {
		fail(w, http.StatusBadRequest, CodeInvalidRequest,
			"this server has no account to sign in to")
		return
	}

	creds, ok := decodeJSON[Credentials](s, w, r)
	if !ok {
		return
	}

	session, err := s.auth.SignIn(w, r, creds.User, creds.Password, creds.Remember)
	switch {
	case errors.Is(err, auth.ErrBadCredentials):
		// One message for a wrong name and a wrong password. Telling them
		// apart would turn the sign-in form into a way to ask whether an
		// account exists.
		fail(w, http.StatusUnauthorized, CodeUnauthorized, "incorrect user name or password")
		return
	case err != nil:
		if wait, is := errors.AsType[*auth.TooManyAttempts](err); is {
			retry := max(wait.RetryAfter.Round(time.Second), time.Second)
			w.Header().Set("Retry-After", strconv.Itoa(int(retry.Seconds())))
			// Said in the API's own words rather than the package's: this
			// sentence is read by whoever is locked out.
			fail(w, http.StatusTooManyRequests, CodeTooManyAttempts,
				"too many attempts; try again in "+retry.String())
			return
		}
		s.failErr(w, err)
		return
	}

	s.log.Info("signed in", "user", session.User, "until", session.Expires)
	writeJSON(w, http.StatusOK, SessionInfo{
		Required:      true,
		Authenticated: true,
		User:          session.User,
		ExpiresAt:     session.Expires,
	})
}

// handleLogout clears the cookie, and says nothing about whether there was
// one: signing out of a session that has already expired is not an error a
// client can do anything with.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.SignOut(w, r)
	w.WriteHeader(http.StatusNoContent)
}
