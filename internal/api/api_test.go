package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegister_RegistersAllOperations boots a fresh gin engine, mounts the
// typed API on it and verifies the OpenAPI spec advertises every operation
// we expect to ship in Phase 1. This is the canary that catches regressions
// like a renamed path or a stray duplicate registration crashing gin at boot.
func TestRegister_RegistersAllOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinContextInjector())

	// Minimal Deps: the smoke test exercises only spec serving and middleware
	// wiring, so nil service/middleware values are fine — the handlers that
	// would dereference them aren't invoked here.
	Register(r, Deps{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/openapi.json: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("openapi spec is not valid JSON: %v", err)
	}

	want := map[string]string{
		"POST /api/v1/auth/login":    "auth-login",
		"POST /api/v1/auth/refresh":  "auth-refresh",
		"POST /api/v1/auth/logout":   "auth-logout",
		"GET /api/v1/profile":        "profile-get",
		"GET /api/v1/tokens":         "tokens-list",
		"POST /api/v1/tokens":        "tokens-create",
		"DELETE /api/v1/tokens/{id}": "tokens-delete",
	}
	for key, opID := range want {
		parts := strings.SplitN(key, " ", 2)
		method, path := strings.ToLower(parts[0]), parts[1]
		methods, ok := spec.Paths[path]
		if !ok {
			t.Errorf("openapi spec missing path %q", path)
			continue
		}
		op, ok := methods[method]
		if !ok {
			t.Errorf("openapi spec missing %s on %q", strings.ToUpper(method), path)
			continue
		}
		if op.OperationID != opID {
			t.Errorf("openapi %s %s operationId = %q, want %q", strings.ToUpper(method), path, op.OperationID, opID)
		}
	}
}

// TestRegister_DocsServed verifies the Stoplight Elements doc UI is mounted
// at the configured path. We only check the response contents enough to
// catch a renamed path or a panic during rendering — we don't try to drive
// the embedded JS.
func TestRegister_DocsServed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GinContextInjector())
	Register(r, Deps{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/docs: status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<elements-api") && !strings.Contains(body, "stoplight") {
		t.Fatalf("expected Stoplight Elements UI markup, got: %q", body[:min(200, len(body))])
	}
}
