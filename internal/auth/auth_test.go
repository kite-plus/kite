package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kite-plus/kite/internal/auth"
)

var frozen = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// newAccount stores an account in a fresh project.
func newAccount(t *testing.T) (string, *auth.Account) {
	t.Helper()
	root := t.TempDir()
	a, err := auth.SetPassword(root, "admin", "correct horse battery")
	if err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	return root, a
}

func TestTheRightPasswordIsAcceptedAndEveryOtherOneIsNot(t *testing.T) {
	_, a := newAccount(t)

	if err := a.Verify("admin", "correct horse battery"); err != nil {
		t.Errorf("the stored password was refused: %v", err)
	}
	for _, c := range []struct{ user, password string }{
		{"admin", "correct horse batter"},
		{"admin", ""},
		{"root", "correct horse battery"},
	} {
		if err := a.Verify(c.user, c.password); !errors.Is(err, auth.ErrBadCredentials) {
			t.Errorf("Verify(%q, %q) = %v, want ErrBadCredentials", c.user, c.password, err)
		}
	}
}

// The file is the one place a password could leak from, so what is in it and
// who can read it are both part of the contract.
func TestTheStoredAccountHoldsNoPasswordAndIsReadableOnlyByItsOwner(t *testing.T) {
	root, _ := newAccount(t)

	path := filepath.Join(root, filepath.FromSlash(auth.File))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", auth.File, err)
	}
	if strings.Contains(string(data), "correct horse battery") {
		t.Error("the password itself is in the file")
	}
	if !strings.Contains(string(data), "$argon2id$") {
		t.Errorf("the file holds no argon2id hash:\n%s", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestAPasswordTooShortToBeWorthHavingIsRefused(t *testing.T) {
	root := t.TempDir()
	if _, err := auth.SetPassword(root, "admin", "short"); err == nil {
		t.Fatal("a five character password was accepted")
	}
	if _, err := auth.Load(root); !errors.Is(err, auth.ErrNoAccount) {
		t.Errorf("Load after a refused password = %v, want ErrNoAccount", err)
	}
}

// A session is a signature rather than a row in memory, which is what lets it
// survive a restart; the same property is why changing the password has to
// end it, since there is no list of sessions to revoke.
func TestASessionSurvivesARestartAndEndsWithThePasswordItWasIssuedUnder(t *testing.T) {
	root, a := newAccount(t)

	token, expires, err := a.Issue(frozen, auth.SessionLifetime)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if !expires.After(frozen) {
		t.Fatalf("expires = %v, want after %v", expires, frozen)
	}

	restarted, err := auth.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	session, err := restarted.Parse(token, frozen.Add(time.Hour))
	if err != nil {
		t.Fatalf("a session did not survive a restart: %v", err)
	}
	if session.User != "admin" {
		t.Errorf("user = %q, want admin", session.User)
	}

	changed, err := auth.SetPassword(root, "admin", "a different password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := changed.Parse(token, frozen.Add(time.Hour)); !errors.Is(err, auth.ErrBadSession) {
		t.Errorf("a session outlived the password it was issued under: %v", err)
	}
}

func TestASessionThatWasEditedOrHasRunOutIsRefused(t *testing.T) {
	_, a := newAccount(t)

	token, _, err := a.Issue(frozen, auth.SessionLifetime)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := a.Parse(token, frozen.Add(auth.SessionLifetime)); !errors.Is(err, auth.ErrBadSession) {
		t.Errorf("an expired session was accepted: %v", err)
	}

	// The payload is the middle segment; changing one character of it must
	// fail the signature rather than change who the session is for.
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d segments, want 3", len(parts))
	}
	for _, forged := range []string{
		parts[0] + "." + parts[1] + "x." + parts[2],
		parts[0] + "." + parts[1] + "." + parts[2] + "x",
		"k1.." + parts[2],
		parts[1] + "." + parts[2],
	} {
		if _, err := a.Parse(forged, frozen); err == nil {
			t.Errorf("a forged session was accepted: %q", forged)
		}
	}
}

// A container sets the environment deliberately and cannot run a command
// inside the image first, so an account left in a mounted volume must not
// quietly outrank it.
func TestTheEnvironmentOutranksTheStoredAccount(t *testing.T) {
	root, _ := newAccount(t)

	t.Setenv("KITE_ADMIN_USER", "operator")
	t.Setenv("KITE_ADMIN_PASSWORD", "from the environment")

	a, err := auth.Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if a.User() != "operator" {
		t.Errorf("user = %q, want operator", a.User())
	}
	if err := a.Verify("operator", "from the environment"); err != nil {
		t.Errorf("the environment password was refused: %v", err)
	}
	if err := a.Verify("admin", "correct horse battery"); !errors.Is(err, auth.ErrBadCredentials) {
		t.Error("the stored account still worked while the environment set one")
	}
}

func TestAProjectWithNoAccountSaysSo(t *testing.T) {
	root := t.TempDir()
	if _, err := auth.Open(root); !errors.Is(err, auth.ErrNoAccount) {
		t.Errorf("Open on an empty project = %v, want ErrNoAccount", err)
	}
	if auth.Configured(root) {
		t.Error("Configured reported an account that was never set")
	}
}

func TestGuessingIsThrottledAndACorrectPasswordStillWaits(t *testing.T) {
	_, a := newAccount(t)

	now := frozen
	g := auth.NewWithClock(a, func() time.Time { return now })

	try := func(password string) error {
		_, err := g.SignIn(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodPost, "/auth/login", nil), "admin", password, false)
		return err
	}

	for i := range 6 {
		if err := try("wrong"); !errors.Is(err, auth.ErrBadCredentials) {
			t.Fatalf("attempt %d = %v, want ErrBadCredentials", i+1, err)
		}
	}

	err := try("correct horse battery")
	wait, refused := errors.AsType[*auth.TooManyAttempts](err)
	if !refused {
		t.Fatalf("the attempt after six failures = %v, want TooManyAttempts", err)
	}
	if wait.RetryAfter <= 0 {
		t.Errorf("RetryAfter = %v, want a delay to report", wait.RetryAfter)
	}

	now = now.Add(wait.RetryAfter)
	if err := try("correct horse battery"); err != nil {
		t.Errorf("the right password was still refused after the wait: %v", err)
	}
}

func TestSigningInSetsACookieTheBrowserWillNotHandToScripts(t *testing.T) {
	_, a := newAccount(t)
	g := auth.NewWithClock(a, func() time.Time { return frozen })

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	if _, err := g.SignIn(rec, r, "admin", "correct horse battery", false); err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != auth.CookieName {
		t.Fatalf("cookies = %v, want one %s", cookies, auth.CookieName)
	}
	c := cookies[0]
	if !c.HttpOnly {
		t.Error("the session cookie is readable from JavaScript")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	// Not remembered means no Max-Age: the cookie goes when the browser does.
	if c.MaxAge != 0 || !c.Expires.IsZero() {
		t.Errorf("an unremembered session was made persistent: %+v", c)
	}

	r.AddCookie(c)
	if _, err := g.Session(r); err != nil {
		t.Errorf("the cookie just issued was not accepted: %v", err)
	}

	out := httptest.NewRecorder()
	g.SignOut(out, r)
	if cleared := out.Result().Cookies(); len(cleared) != 1 || cleared[0].MaxAge >= 0 {
		t.Errorf("signing out did not clear the cookie: %v", cleared)
	}
}

func TestOnlyAnAddressThisMachineAnswersForCountsAsLoopback(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:1717": true,
		"127.2.3.4:1717": true,
		"localhost:1717": true,
		"[::1]:1717":     true,
		"0.0.0.0:1717":   false,
		":1717":          false,
		"203.0.113.5:80": false,
		// A name is not an address: it can be pointed anywhere, so it is
		// treated as public rather than resolved and trusted.
		"studio.example.com:80": false,
	} {
		if got := auth.Loopback(addr); got != want {
			t.Errorf("Loopback(%q) = %v, want %v", addr, got, want)
		}
	}
}
