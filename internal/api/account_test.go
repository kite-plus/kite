package api_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/auth"
)

// keeping gives a test server a keeper for its account, over a stored one
// when stored is true, as a server listening on addr would have.
func keeping(t *testing.T, root, addr string, stored bool) func(*api.Options) {
	t.Helper()
	var account *auth.Account
	if stored {
		var err error
		if account, err = auth.SetPassword(root, "admin", password); err != nil {
			t.Fatalf("SetPassword: %v", err)
		}
	}
	guard := auth.New(account)
	return func(o *api.Options) {
		o.Auth = guard
		o.Account = auth.NewKeeper(root, guard, addr)
	}
}

const localAddr = "127.0.0.1:1717"

func accountOf(t *testing.T, rec *httptest.ResponseRecorder) api.AccountInfo {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var info api.AccountInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode account: %v\nbody: %s", err, rec.Body.String())
	}
	return info
}

// sessionCookie is the session a response set, or nil.
func sessionCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	return nil
}

func refusal(t *testing.T, rec *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, status, rec.Body.String())
	}
	got := failureOf(t, rec)
	if got.Code != code || got.Field != field {
		t.Errorf("refusal = %+v, want code %q on field %q", got, code, field)
	}
}

func TestAnOpenStudioCanBeGivenAPasswordFromTheStudio(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, keeping(t, root, localAddr, false))

	info := accountOf(t, call(t, h, http.MethodGet, api.Prefix+"/account", nil))
	if info.Protected || !info.Editable || info.Removable || info.Session != nil || info.User != "" {
		t.Fatalf("open studio = %+v, want unprotected, editable, nothing to remove", info)
	}

	short := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{Password: "short"})
	refusal(t, short, http.StatusBadRequest, api.CodeInvalidRequest, "password")

	rec := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{Password: password})
	info = accountOf(t, rec)
	if !info.Protected || info.User != "admin" || info.Source != "file" || info.Session == nil {
		t.Fatalf("after setting a password = %+v, want admin signed in", info)
	}
	cookie := sessionCookie(rec)
	if cookie == nil {
		t.Fatal("setting the password did not sign this browser in")
	}

	// The running server asks for it at once, not at the next start.
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /contents with no session: status = %d, want 401", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, cookie); rec.Code != http.StatusOK {
		t.Errorf("GET /contents with the new session: status = %d, want 200", rec.Code)
	}
	if !auth.Configured(root) {
		t.Error("the account was not stored")
	}
}

func TestChangingThePasswordKeepsThisBrowserSignedInAndNoOtherOne(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, keeping(t, root, localAddr, true))

	here, there := signIn(t, h), signIn(t, h)

	wrong := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: "not it", Password: "a whole new password"}, here)
	// Not 401: that would tell the studio its session had ended.
	refusal(t, wrong, http.StatusForbidden, api.CodeWrongPassword, "current_password")

	same := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: password, User: "admin"}, here)
	refusal(t, same, http.StatusBadRequest, api.CodeInvalidRequest, "")

	rec := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: password, Password: "a whole new password"}, here)
	info := accountOf(t, rec)
	renewed := sessionCookie(rec)
	if renewed == nil || info.Session == nil {
		t.Fatalf("the browser that changed the password was not issued a session again: %+v", info)
	}

	for name, c := range map[string]*http.Cookie{"the old cookie": here, "another browser": there} {
		if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, c); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s after the change: status = %d, want 401", name, rec.Code)
		}
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, renewed); rec.Code != http.StatusOK {
		t.Errorf("the renewed session: status = %d, want 200", rec.Code)
	}

	old := call(t, h, http.MethodPost, api.Prefix+"/auth/login", api.Credentials{User: "admin", Password: password})
	if old.Code != http.StatusUnauthorized {
		t.Errorf("signing in with the old password: status = %d, want 401", old.Code)
	}
	fresh := call(t, h, http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "admin", Password: "a whole new password"})
	if fresh.Code != http.StatusOK {
		t.Errorf("signing in with the new password: status = %d, want 200", fresh.Code)
	}
}

func TestRenamingIsConfirmedAndReportedBack(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, keeping(t, root, localAddr, true))
	cookie := signIn(t, h)

	blank := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: password, User: "has\ttab"}, cookie)
	refusal(t, blank, http.StatusBadRequest, api.CodeInvalidRequest, "user")

	rec := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: password, User: " owner "}, cookie)
	if info := accountOf(t, rec); info.User != "owner" {
		t.Fatalf("user = %q, want owner", info.User)
	}
	info := accountOf(t, call(t, h, http.MethodGet, api.Prefix+"/account", nil, sessionCookie(rec)))
	if info.User != "owner" {
		t.Errorf("GET /account after the rename: user = %q, want owner", info.User)
	}
	login := call(t, h, http.MethodPost, api.Prefix+"/auth/login", api.Credentials{User: "owner", Password: password})
	if login.Code != http.StatusOK {
		t.Errorf("signing in under the new name: status = %d, want 200", login.Code)
	}
}

func TestSigningOutEverywhereElseKeepsThisSessionAsItWas(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, keeping(t, root, localAddr, true))

	login := call(t, h, http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "admin", Password: password, Remember: true})
	here, there := sessionCookie(login), signIn(t, h)

	rec := call(t, h, http.MethodDelete, api.Prefix+"/account/sessions", nil, here)
	info := accountOf(t, rec)
	renewed := sessionCookie(rec)
	if renewed == nil || info.Session == nil || !info.Session.Remembered {
		t.Fatalf("session after signing out elsewhere = %+v, want this one kept and remembered", info.Session)
	}
	if renewed.MaxAge <= 0 {
		t.Errorf("a remembered session came back as a browser-window cookie: %+v", renewed)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, there); rec.Code != http.StatusUnauthorized {
		t.Errorf("another browser afterwards: status = %d, want 401", rec.Code)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, renewed); rec.Code != http.StatusOK {
		t.Errorf("this browser afterwards: status = %d, want 200", rec.Code)
	}
}

func TestThePasswordComesOffOnlyWhereNobodyElseCanReachTheStudio(t *testing.T) {
	public := newProject(t, 1)
	h, _ := newWritableServer(t, public, keeping(t, public, "0.0.0.0:1717", true))
	cookie := signIn(t, h)
	if info := accountOf(t, call(t, h, http.MethodGet, api.Prefix+"/account", nil, cookie)); info.Removable {
		t.Error("a public server offers to remove its password")
	}
	rec := call(t, h, http.MethodDelete, api.Prefix+"/account/credentials",
		api.PasswordConfirmation{CurrentPassword: password}, cookie)
	refusal(t, rec, http.StatusConflict, api.CodePasswordRequired, "")

	local := newProject(t, 1)
	h, _ = newWritableServer(t, local, keeping(t, local, localAddr, true))
	cookie = signIn(t, h)
	if info := accountOf(t, call(t, h, http.MethodGet, api.Prefix+"/account", nil, cookie)); !info.Removable {
		t.Error("a local server does not offer to remove its password")
	}
	wrong := call(t, h, http.MethodDelete, api.Prefix+"/account/credentials",
		api.PasswordConfirmation{CurrentPassword: "not it"}, cookie)
	refusal(t, wrong, http.StatusForbidden, api.CodeWrongPassword, "current_password")

	rec = call(t, h, http.MethodDelete, api.Prefix+"/account/credentials",
		api.PasswordConfirmation{CurrentPassword: password}, cookie)
	if info := accountOf(t, rec); info.Protected || info.Session != nil {
		t.Errorf("after removal = %+v, want an open studio", info)
	}
	if c := sessionCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Errorf("the session cookie was not cleared: %+v", c)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil); rec.Code != http.StatusOK {
		t.Errorf("GET /contents on the opened studio: status = %d, want 200", rec.Code)
	}
}

func TestAnAccountFromTheEnvironmentIsChangedThere(t *testing.T) {
	t.Setenv("KITE_ADMIN_USER", "operator")
	t.Setenv("KITE_ADMIN_PASSWORD", password)
	root := newProject(t, 1)
	account, err := auth.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	guard := auth.New(account)
	h, _ := newWritableServer(t, root, func(o *api.Options) {
		o.Auth = guard
		o.Account = auth.NewKeeper(root, guard, localAddr)
	})

	login := call(t, h, http.MethodPost, api.Prefix+"/auth/login", api.Credentials{User: "operator", Password: password})
	cookie := sessionCookie(login)

	info := accountOf(t, call(t, h, http.MethodGet, api.Prefix+"/account", nil, cookie))
	if info.Source != "environment" || info.Editable || info.Removable || !info.ProfileEditable {
		t.Errorf("account = %+v, want an environment account whose profile alone can change", info)
	}
	rec := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: password, Password: "another password"}, cookie)
	refusal(t, rec, http.StatusConflict, api.CodeAccountFixed, "")
	rec = call(t, h, http.MethodDelete, api.Prefix+"/account/sessions", nil, cookie)
	refusal(t, rec, http.StatusConflict, api.CodeAccountFixed, "")
}

func TestAReadOnlyServerChangesNoAccount(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newServer(t, root, keeping(t, root, localAddr, false))

	info := accountOf(t, call(t, h, http.MethodGet, api.Prefix+"/account", nil))
	if info.Editable || info.ProfileEditable {
		t.Errorf("a read-only server offers changes: %+v", info)
	}
	rec := call(t, h, http.MethodPut, api.Prefix+"/account/profile", api.Profile{Name: "Ada"})
	refusal(t, rec, http.StatusMethodNotAllowed, api.CodeReadOnly, "")
	rec = call(t, h, http.MethodPut, api.Prefix+"/account/credentials", api.CredentialsChange{Password: password})
	refusal(t, rec, http.StatusMethodNotAllowed, api.CodeReadOnly, "")
	if auth.Configured(root) {
		t.Error("a read-only server stored an account")
	}
}

// uploadAvatar sends a picture the way the studio's form does.
func uploadAvatar(t *testing.T, h http.Handler, name string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	form := multipart.NewWriter(&buf)
	part, err := form.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	r := httptest.NewRequest(http.MethodPut, api.Prefix+"/account/avatar", &buf)
	r.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestTheProfileAndItsPictureAreStoredAndServed(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, keeping(t, root, localAddr, false))

	info := accountOf(t, call(t, h, http.MethodPut, api.Prefix+"/account/profile",
		api.Profile{Name: " Ada Lovelace ", Email: "ada@example.com"}))
	if info.Profile != (api.Profile{Name: "Ada Lovelace", Email: "ada@example.com"}) {
		t.Errorf("profile = %+v, want it trimmed and stored", info.Profile)
	}
	bad := call(t, h, http.MethodPut, api.Prefix+"/account/profile", api.Profile{Email: "not an address"})
	refusal(t, bad, http.StatusBadRequest, api.CodeInvalidRequest, "email")

	var pic bytes.Buffer
	if err := png.Encode(&pic, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	info = accountOf(t, uploadAvatar(t, h, "me.png", pic.Bytes()))
	if !strings.HasPrefix(info.Avatar, api.Prefix+"/account/avatar?v=") {
		t.Fatalf("avatar = %q, want a versioned address", info.Avatar)
	}

	rec := call(t, h, http.MethodGet, info.Avatar, nil)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" ||
		!bytes.Equal(rec.Body.Bytes(), pic.Bytes()) {
		t.Fatalf("GET the picture: status %d, type %q, %d bytes",
			rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("the picture is served without nosniff")
	}
	r := httptest.NewRequest(http.MethodGet, info.Avatar, nil)
	r.Header.Set("If-None-Match", rec.Header().Get("ETag"))
	again := httptest.NewRecorder()
	h.ServeHTTP(again, r)
	if again.Code != http.StatusNotModified {
		t.Errorf("GET the same picture again: status = %d, want 304", again.Code)
	}

	svg := uploadAvatar(t, h, "me.png",
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`))
	refusal(t, svg, http.StatusUnsupportedMediaType, api.CodeInvalidRequest, "file")

	info = accountOf(t, call(t, h, http.MethodDelete, api.Prefix+"/account/avatar", nil))
	if info.Avatar != "" {
		t.Errorf("avatar after removal = %q, want none", info.Avatar)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/account/avatar", nil); rec.Code != http.StatusNotFound {
		t.Errorf("GET a removed picture: status = %d, want 404", rec.Code)
	}
}

func TestConfirmingAChangeIsThrottledLikeSigningIn(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newWritableServer(t, root, keeping(t, root, localAddr, true))
	cookie := signIn(t, h)

	for range 6 {
		call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
			api.CredentialsChange{CurrentPassword: "guess", Password: "a whole new password"}, cookie)
	}
	rec := call(t, h, http.MethodPut, api.Prefix+"/account/credentials",
		api.CredentialsChange{CurrentPassword: password, Password: "a whole new password"}, cookie)
	refusal(t, rec, http.StatusTooManyRequests, api.CodeTooManyAttempts, "")
	if rec.Header().Get("Retry-After") == "" {
		t.Error("a throttled change does not say when to try again")
	}
}
