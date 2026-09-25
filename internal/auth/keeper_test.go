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

const oldPassword = "correct horse battery"

// signedIn returns a guard over a stored account and a request carrying a
// session for it.
func signedIn(t *testing.T, remember bool) (string, *auth.Guard, *http.Request) {
	t.Helper()
	root, a := newAccount(t)
	g := auth.NewWithClock(a, func() time.Time { return frozen })

	rec := httptest.NewRecorder()
	if _, err := g.SignIn(rec, httptest.NewRequest(http.MethodPost, "/auth/login", nil),
		"admin", oldPassword, remember); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	return root, g, withCookies(rec)
}

// withCookies is a request carrying the cookies a response set.
func withCookies(rec *httptest.ResponseRecorder) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rec.Result().Cookies() {
		r.AddCookie(c)
	}
	return r
}

// A session is a signature over the name and the password, so a change to
// either ends every session; the browser that made the change is issued a
// new one, running out when its old one would have.
func TestChangingThePasswordEndsEverySessionButTheOneRenewed(t *testing.T) {
	root, g, r := signedIn(t, false)
	k := auth.NewKeeper(root, g, "127.0.0.1:1717")

	before, err := g.Session(r)
	if err != nil {
		t.Fatalf("Session before the change: %v", err)
	}
	current, err := g.Confirm(oldPassword)
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if _, err := k.Change(current, "", "a new password entirely"); err != nil {
		t.Fatalf("Change: %v", err)
	}

	if _, err := g.Session(r); !errors.Is(err, auth.ErrBadSession) {
		t.Errorf("a session outlived the password change: %v", err)
	}

	rec := httptest.NewRecorder()
	renewed, err := g.Renew(rec, r, before)
	if err != nil {
		t.Fatalf("Renew: %v", err)
	}
	if !renewed.Expires.Equal(before.Expires) {
		t.Errorf("renewed session expires %v, want %v as before", renewed.Expires, before.Expires)
	}
	if _, err := g.Session(withCookies(rec)); err != nil {
		t.Errorf("the renewed session was refused: %v", err)
	}

	// What a restart reads is what the running server now checks.
	stored, err := auth.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := stored.Verify("admin", "a new password entirely"); err != nil {
		t.Errorf("the stored account kept the old password: %v", err)
	}
	if _, err := g.SignIn(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil),
		"admin", oldPassword, false); !errors.Is(err, auth.ErrBadCredentials) {
		t.Errorf("the old password still signs in: %v", err)
	}
}

func TestRenamingKeepsThePassword(t *testing.T) {
	root, g, r := signedIn(t, false)
	k := auth.NewKeeper(root, g, "127.0.0.1:1717")

	current, err := g.Confirm(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	next, err := k.Change(current, "owner", "")
	if err != nil {
		t.Fatalf("Change: %v", err)
	}
	if next.User() != "owner" || g.User() != "owner" {
		t.Errorf("user = %q, guard user = %q, want owner", next.User(), g.User())
	}
	if err := next.Verify("owner", oldPassword); err != nil {
		t.Errorf("the password did not survive the rename: %v", err)
	}
	if _, err := g.Session(r); !errors.Is(err, auth.ErrBadSession) {
		t.Errorf("a session for the old name was still accepted: %v", err)
	}
}

// Signing out everywhere changes nothing about how anybody signs in: only the
// secret the sessions were signed with.
func TestRotatingTheSecretEndsEverySessionAndKeepsTheSignIn(t *testing.T) {
	root, g, r := signedIn(t, true)
	k := auth.NewKeeper(root, g, "127.0.0.1:1717")
	session, err := g.Session(r)
	if err != nil {
		t.Fatal(err)
	}
	if !session.Remember {
		t.Fatal("a remembered sign-in did not say so")
	}

	before, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(auth.File)))
	if err != nil {
		t.Fatal(err)
	}
	next, err := k.Rotate(g.Account())
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(auth.File)))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) == string(after) {
		t.Error("the stored account did not change")
	}

	if _, err := g.Session(r); !errors.Is(err, auth.ErrBadSession) {
		t.Errorf("a session outlived the secret it was signed with: %v", err)
	}
	if err := next.Verify("admin", oldPassword); err != nil {
		t.Errorf("rotating the secret changed the password: %v", err)
	}

	// Kept past the browser window before, so kept past it after.
	rec := httptest.NewRecorder()
	if _, err := g.Renew(rec, r, session); err != nil {
		t.Fatal(err)
	}
	c := rec.Result().Cookies()
	if len(c) != 1 || c[0].MaxAge <= 0 {
		t.Errorf("a remembered session came back as %+v, want a persistent cookie", c)
	}
}

func TestAnAccountFromTheEnvironmentIsChangedThereAndNotHere(t *testing.T) {
	t.Setenv("KITE_ADMIN_PASSWORD", "from the environment")
	root := t.TempDir()
	a, err := auth.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	g := auth.New(a)
	k := auth.NewKeeper(root, g, "127.0.0.1:1717")

	if _, err := k.Change(a, "", "another password"); !errors.Is(err, auth.ErrFixedAccount) {
		t.Errorf("Change = %v, want ErrFixedAccount", err)
	}
	if _, err := k.Rotate(a); !errors.Is(err, auth.ErrFixedAccount) {
		t.Errorf("Rotate = %v, want ErrFixedAccount", err)
	}
	if err := k.Remove(a); !errors.Is(err, auth.ErrFixedAccount) {
		t.Errorf("Remove = %v, want ErrFixedAccount", err)
	}
	if _, err := auth.Load(root); !errors.Is(err, auth.ErrNoAccount) {
		t.Errorf("a refused change left an account file behind: %v", err)
	}
}

// Two tabs that confirmed the same password must not both apply a change:
// the second would be written over the first without anybody seeing it.
func TestAChangeToAnAccountThatWasReplacedIsRefused(t *testing.T) {
	root, g, _ := signedIn(t, false)
	k := auth.NewKeeper(root, g, "127.0.0.1:1717")

	stale, err := g.Confirm(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := k.Change(stale, "", "the first new password"); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Change(stale, "", "the second new password"); !errors.Is(err, auth.ErrAccountChanged) {
		t.Errorf("a change against a replaced account = %v, want ErrAccountChanged", err)
	}
}

func TestThePasswordStaysOnAServerOthersCanReach(t *testing.T) {
	root, g, _ := signedIn(t, false)

	public := auth.NewKeeper(root, g, "0.0.0.0:1717")
	if err := public.Remove(g.Account()); !errors.Is(err, auth.ErrMustStayGuarded) {
		t.Fatalf("Remove on a public server = %v, want ErrMustStayGuarded", err)
	}
	if !g.Required() || !auth.Configured(root) {
		t.Fatal("a refused removal still opened the studio")
	}

	local := auth.NewKeeper(root, g, "127.0.0.1:1717")
	if err := local.Remove(g.Account()); err != nil {
		t.Fatalf("Remove on localhost: %v", err)
	}
	if g.Required() {
		t.Error("the guard still asks for a password")
	}
	if auth.Configured(root) {
		t.Error("the account file is still there")
	}
}

func TestAnOpenStudioIsGivenAnAccountOnce(t *testing.T) {
	root := t.TempDir()
	g := auth.New(nil)
	k := auth.NewKeeper(root, g, "127.0.0.1:1717")

	if _, err := k.Create("owner", "short"); err == nil {
		t.Fatal("a five character password was accepted")
	}
	if g.Required() {
		t.Fatal("a refused account was installed")
	}

	a, err := k.Create("owner", oldPassword)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !g.Required() || g.User() != "owner" {
		t.Errorf("guard required = %v, user = %q, want owner", g.Required(), g.User())
	}
	if err := a.Verify("owner", oldPassword); err != nil {
		t.Error(err)
	}
	if _, err := k.Create("someone", "another password"); !errors.Is(err, auth.ErrAlreadyConfigured) {
		t.Errorf("a second Create = %v, want ErrAlreadyConfigured", err)
	}
}

// A borrowed browser must not be a way to guess at the password with no
// limit, so confirming one shares the allowance signing in has.
func TestConfirmingThePasswordIsThrottledLikeSigningIn(t *testing.T) {
	_, a := newAccount(t)
	now := frozen
	g := auth.NewWithClock(a, func() time.Time { return now })

	for i := range 6 {
		if _, err := g.Confirm("wrong"); !errors.Is(err, auth.ErrBadCredentials) {
			t.Fatalf("attempt %d = %v, want ErrBadCredentials", i+1, err)
		}
	}
	if _, err := g.Confirm(oldPassword); err == nil {
		t.Fatal("the right password went through straight after six wrong ones")
	} else if _, throttled := errors.AsType[*auth.TooManyAttempts](err); !throttled {
		t.Fatalf("Confirm = %v, want TooManyAttempts", err)
	}
	_, err := g.SignIn(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil),
		"admin", oldPassword, false)
	if _, throttled := errors.AsType[*auth.TooManyAttempts](err); !throttled {
		t.Errorf("signing in after failed confirmations = %v, want TooManyAttempts", err)
	}
}

func TestANameOrPasswordThatCannotBeUsedIsRefusedWithAReason(t *testing.T) {
	for _, user := range []string{"", "  ", " admin", "admin ", "ad\tmin", strings.Repeat("a", auth.MaxUserLength+1)} {
		if err := auth.CheckUser(user); err == nil {
			t.Errorf("CheckUser(%q) accepted it", user)
		}
	}
	for _, user := range []string{"admin", "Ada Lovelace", "作者"} {
		if err := auth.CheckUser(user); err != nil {
			t.Errorf("CheckUser(%q) = %v", user, err)
		}
	}

	for _, password := range []string{"", "seven77", strings.Repeat("a", auth.MaxPasswordLength+1)} {
		if err := auth.CheckPassword(password); err == nil {
			t.Errorf("CheckPassword accepted %d characters", len([]rune(password)))
		}
	}
	// Characters, not bytes: eight Chinese characters are eight.
	if err := auth.CheckPassword("一二三四五六七八"); err != nil {
		t.Errorf("CheckPassword(eight characters) = %v", err)
	}
}
