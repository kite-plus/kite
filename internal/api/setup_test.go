package api_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kite-plus/kite/internal/api"
	"github.com/kite-plus/kite/internal/auth"
	"github.com/kite-plus/kite/internal/setup"
)

// newUnconfiguredServer is a writable server that nobody has set up yet,
// which is what a container is on its first start.
func newUnconfiguredServer(t *testing.T, root string) (http.Handler, *setup.Flow, *auth.Guard) {
	t.Helper()

	guard := auth.New(nil)
	flow, err := setup.New(root, guard)
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	h, _ := newWritableServer(t, root, func(o *api.Options) {
		o.Auth, o.Setup = guard, flow
	})
	return h, flow, guard
}

func setupBody() map[string]any {
	return map[string]any{
		"user":     "editor",
		"password": "correct horse battery",
		"site": map[string]any{
			"title":       "Notebook",
			"description": "Notes on things",
			"base_url":    "https://notes.example.com",
			"language":    "zh-CN",
		},
	}
}

// A server waiting to be set up is a server anyone can reach, so it must not
// answer questions about what it holds.
func TestNothingButSetupAnswersUntilAServerHasAnOwner(t *testing.T) {
	h, _, _ := newUnconfiguredServer(t, newProject(t, 3))

	for _, path := range []string{"/contents", "/site", "/settings", "/taxonomies", "/publish"} {
		rec := send(t, h, http.MethodGet, api.Prefix+path, nil, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("GET %s: status = %d, want 403", path, rec.Code)
		}
		if got := decode[api.ErrorBody](t, rec).Error.Code; got != api.CodeSetupRequired {
			t.Errorf("GET %s: code = %q, want %q", path, got, api.CodeSetupRequired)
		}
	}

	// The way in is open, or there would be no way in.
	if rec := send(t, h, http.MethodGet, api.Prefix+"/setup", nil, nil); rec.Code != http.StatusOK {
		t.Errorf("GET /setup: status = %d, want 200", rec.Code)
	}
	state := get[api.SetupState](t, h, api.Prefix+"/setup", http.StatusOK)
	if !state.Required {
		t.Errorf("state = %+v, want setup to be required", state)
	}
	// The form starts from what the project already says about itself.
	if state.Site == nil || state.Site.Title != "Field Notes" {
		t.Errorf("site = %+v, want the project's own title", state.Site)
	}
}

func TestSetupRefusesAPasswordTooShortToKeepAndAnAddressThatIsNotOne(t *testing.T) {
	root := newProject(t, 1)
	h, flow, guard := newUnconfiguredServer(t, root)

	short := setupBody()
	short["password"] = "short"
	rec := send(t, h, http.MethodPost, api.Prefix+"/setup", short, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("short password: status = %d, want 400\n%s", rec.Code, rec.Body)
	}
	if got := decode[api.ErrorBody](t, rec).Error.Field; got != "password" {
		t.Errorf("field = %q, want password", got)
	}

	bad := setupBody()
	bad["site"] = map[string]any{"title": "T", "base_url": "not a url", "language": "en"}
	rec = send(t, h, http.MethodPost, api.Prefix+"/setup", bad, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad base url: status = %d, want 400\n%s", rec.Code, rec.Body)
	}

	// None of that half-installed anything.
	if guard.Required() || !flow.Pending() {
		t.Error("a refused attempt left the server configured")
	}
	// Checked against a value only the form could have introduced: the
	// fixture already calls itself Field Notes.
	if body, _ := os.ReadFile(filepath.Join(root, "kite.yaml")); strings.Contains(string(body), "notes.example.com") {
		t.Error("a refused attempt wrote the site settings anyway")
	}
}

func TestSetupWritesTheSiteCreatesTheAccountAndSignsTheBrowserIn(t *testing.T) {
	root := newProject(t, 1)
	h, flow, guard := newUnconfiguredServer(t, root)

	rec := send(t, h, http.MethodPost, api.Prefix+"/setup", setupBody(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body)
	}

	session := decode[api.SessionInfo](t, rec)
	if !session.Required || !session.Authenticated || session.User != "editor" {
		t.Errorf("session = %+v, want an authenticated editor", session)
	}
	// Signed in on the spot, or whoever just chose a password would be asked
	// for it again one field below where they chose it.
	if cookie := rec.Header().Get("Set-Cookie"); !strings.Contains(cookie, auth.CookieName) {
		t.Errorf("no session cookie was set: %q", cookie)
	}

	// Every field of the form reached the file, not just the last one.
	body, err := os.ReadFile(filepath.Join(root, "kite.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"title: Notebook",
		"description: Notes on things",
		"baseURL: https://notes.example.com",
		"language: zh-CN",
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("kite.yaml is missing %q:\n%s", want, body)
		}
	}

	if !guard.Required() || guard.User() != "editor" {
		t.Error("the studio is still open after being set up")
	}
	if flow.Pending() {
		t.Error("setup is still pending after it finished")
	}
	if state := get[api.SetupState](t, h, api.Prefix+"/setup", http.StatusOK); state.Required {
		t.Error("a server that was set up still asks to be")
	}

	// A second attempt cannot take the server over, and the rest of the API
	// now wants a session rather than refusing outright.
	rec = send(t, h, http.MethodPost, api.Prefix+"/setup", setupBody(), nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("a second setup: status = %d, want 409", rec.Code)
	}
	rec = send(t, h, http.MethodGet, api.Prefix+"/contents", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /contents after setup: status = %d, want 401", rec.Code)
	}
}

// A description is optional, and an optional question nobody answered should
// not become a key in the file that reads like a decision.
func TestAnEmptyDescriptionIsLeftOutOfTheConfiguration(t *testing.T) {
	root := newProject(t, 1)
	h, _, _ := newUnconfiguredServer(t, root)

	body := setupBody()
	body["site"] = map[string]any{
		"title":    "Notebook",
		"base_url": "https://notes.example.com",
		"language": "en",
	}
	if rec := send(t, h, http.MethodPost, api.Prefix+"/setup", body, nil); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200\n%s", rec.Code, rec.Body)
	}

	stored, err := os.ReadFile(filepath.Join(root, "kite.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), `description: ""`) {
		t.Errorf("an unanswered description was written anyway:\n%s", stored)
	}
}

// Every other server -- a local preview, one with a password already -- is
// untouched by any of this.
func TestAServerWithNoSetupFlowBehavesExactlyAsBefore(t *testing.T) {
	h, _ := newServer(t, newProject(t, 2))

	if rec := send(t, h, http.MethodGet, api.Prefix+"/contents", nil, nil); rec.Code != http.StatusOK {
		t.Errorf("GET /contents: status = %d, want 200", rec.Code)
	}
	if state := get[api.SetupState](t, h, api.Prefix+"/setup", http.StatusOK); state.Required {
		t.Error("a server with nothing to set up asks to be set up")
	}
}
