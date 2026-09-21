package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/auth"
)

const password = "correct horse battery"

// guarded puts an account in front of a test server.
func guarded(t *testing.T, root string) func(*api.Options) {
	t.Helper()
	account, err := auth.SetPassword(root, "admin", password)
	if err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	return func(o *api.Options) { o.Auth = auth.New(account) }
}

// call issues one request, carrying whatever cookies it is given.
func call(t *testing.T, h http.Handler, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(method, path, nil)
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// failureOf reads the machine-readable code out of a refusal.
func failureOf(t *testing.T, rec *httptest.ResponseRecorder) api.ErrorDetail {
	t.Helper()
	var body api.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v\nbody: %s", err, rec.Body.String())
	}
	return body.Error
}

func sessionOf(t *testing.T, rec *httptest.ResponseRecorder) api.SessionInfo {
	t.Helper()
	var info api.SessionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode session: %v\nbody: %s", err, rec.Body.String())
	}
	return info
}

// signIn returns the cookie a successful sign-in hands back.
func signIn(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	rec := call(t, h, http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "admin", Password: password})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			return c
		}
	}
	t.Fatal("login returned no session cookie")
	return nil
}

func TestAGuardedServerAnswersNothingUntilSomebodySignsIn(t *testing.T) {
	root := newProject(t, 3)
	h, _ := newServer(t, root, guarded(t, root))

	for _, path := range []string{"/contents", "/site", "/settings", "/taxonomies"} {
		rec := call(t, h, http.MethodGet, api.Prefix+path, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s: status = %d, want 401", path, rec.Code)
		}
		if code := failureOf(t, rec).Code; code != api.CodeUnauthorized {
			t.Errorf("GET %s: code = %q, want %q", path, code, api.CodeUnauthorized)
		}
	}

	// An endpoint that does not exist must not answer differently from one
	// that does, or the door becomes a way to read the map.
	rec := call(t, h, http.MethodGet, api.Prefix+"/nothing-here", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET an unknown endpoint: status = %d, want 401", rec.Code)
	}

	// The way in, and the description of it, stay open.
	info := sessionOf(t, call(t, h, http.MethodGet, api.Prefix+"/auth/session", nil))
	if !info.Required || info.Authenticated {
		t.Errorf("session = %+v, want required and not authenticated", info)
	}
	if info.User != "" {
		t.Errorf("an anonymous caller was told the account name: %q", info.User)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+api.OpenAPIPath, nil); rec.Code != http.StatusOK {
		t.Errorf("GET the description: status = %d, want 200", rec.Code)
	}
}

func TestSigningInOpensTheApiAndSigningOutClosesItAgain(t *testing.T) {
	root := newProject(t, 3)
	h, _ := newServer(t, root, guarded(t, root))

	cookie := signIn(t, h)
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, cookie); rec.Code != http.StatusOK {
		t.Fatalf("GET /contents with a session: status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}

	info := sessionOf(t, call(t, h, http.MethodGet, api.Prefix+"/auth/session", nil, cookie))
	if !info.Authenticated || info.User != "admin" {
		t.Errorf("session = %+v, want admin signed in", info)
	}
	if info.ExpiresAt.IsZero() {
		t.Error("a signed-in session does not say when it runs out")
	}

	out := call(t, h, http.MethodPost, api.Prefix+"/auth/logout", nil, cookie)
	if out.Code != http.StatusNoContent {
		t.Fatalf("logout: status = %d, want 204", out.Code)
	}
	cleared := out.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Value != "" || cleared[0].MaxAge >= 0 {
		t.Fatalf("logout did not clear the cookie: %v", cleared)
	}
	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil, cleared[0]); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /contents after signing out: status = %d, want 401", rec.Code)
	}
}

// A sign-in form that answered differently for an unknown name would be a way
// to ask whether an account exists.
func TestAWrongNameAndAWrongPasswordAreRefusedTheSameWay(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newServer(t, root, guarded(t, root))

	badPassword := call(t, h, http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "admin", Password: "not it"})
	badName := call(t, h, http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "someone", Password: password})

	for _, rec := range []*httptest.ResponseRecorder{badPassword, badName} {
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401\nbody: %s", rec.Code, rec.Body.String())
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Error("a refused sign-in still set a cookie")
		}
	}
	if badPassword.Body.String() != badName.Body.String() {
		t.Errorf("the two refusals differ:\n%s\n%s", badPassword.Body, badName.Body)
	}
}

// A cookie travels with a request whoever caused it, so the session alone
// cannot be what decides a write.
func TestARequestFromAnotherSiteIsRefusedEvenWithASession(t *testing.T) {
	root := newProject(t, 1)
	h, _ := newServer(t, root, guarded(t, root))
	cookie := signIn(t, h)

	crossSite := func(method, path string, body any) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Sec-Fetch-Site", "cross-site")
		r.AddCookie(cookie)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec
	}

	rec := crossSite(http.MethodPost, api.Prefix+"/contents", api.Draft{Kind: "post", Title: "Theirs"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("a cross-site write: status = %d, want 403\nbody: %s", rec.Code, rec.Body.String())
	}
	if code := failureOf(t, rec).Code; code != api.CodeCrossOrigin {
		t.Errorf("code = %q, want %q", code, api.CodeCrossOrigin)
	}

	// Signing in is itself a write: a form on another site must not be able
	// to put its own session into somebody's browser.
	login := crossSite(http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "admin", Password: password})
	if login.Code != http.StatusForbidden {
		t.Errorf("a cross-site sign-in: status = %d, want 403", login.Code)
	}

	// The same request from the admin itself gets through to the handler,
	// which refuses it for its own reason: this server has no writer.
	r := httptest.NewRequest(http.MethodPost, api.Prefix+"/contents", bytes.NewReader([]byte(`{"kind":"post"}`)))
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.AddCookie(cookie)
	same := httptest.NewRecorder()
	h.ServeHTTP(same, r)
	if code := failureOf(t, same).Code; code != api.CodeReadOnly {
		t.Errorf("a same-origin write: code = %q, want %q", code, api.CodeReadOnly)
	}
}

func TestAServerWithNoAccountAsksForNothing(t *testing.T) {
	h, _ := newServer(t, newProject(t, 3))

	if rec := call(t, h, http.MethodGet, api.Prefix+"/contents", nil); rec.Code != http.StatusOK {
		t.Errorf("GET /contents on an open server: status = %d, want 200", rec.Code)
	}

	info := sessionOf(t, call(t, h, http.MethodGet, api.Prefix+"/auth/session", nil))
	if info.Required || info.Authenticated {
		t.Errorf("session = %+v, want neither required nor authenticated", info)
	}

	// There is nothing to sign in to, and saying so is better than refusing
	// the password as though it were wrong.
	rec := call(t, h, http.MethodPost, api.Prefix+"/auth/login",
		api.Credentials{User: "admin", Password: password})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("login on an open server: status = %d, want 400", rec.Code)
	}
}
