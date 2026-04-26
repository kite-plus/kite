package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
	"gorm.io/gorm"
)

// integrationCounter ensures every test case binds to its own in-memory
// sqlite namespace; otherwise concurrent runs would clobber each other's
// users table.
var integrationCounter int64

// TestIntegration_LoginProfileTokenRoundTrip drives the typed API end-to-end
// against a real AuthService backed by sqlite. It nails down the contract
// every desktop / mobile client will rely on:
//
//   - Login returns an envelope with code=0 and the token pair.
//   - The X-Kite-API-Version header is stamped on every response.
//   - GET /api/v1/profile honours the Authorization: Bearer header.
//   - PAT minting returns the plaintext exactly once and lets the caller
//     immediately see the token in /tokens.
//   - DELETE /tokens/{id} removes only the caller's own token.
//   - Bad credentials and missing tokens come back as the {code,message,data}
//     error envelope, not as huma's default RFC 7807 shape.
func TestIntegration_LoginProfileTokenRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, authSvc := newIntegrationServer(t)

	// Seed an active user via the same code path the gin /auth/register handler
	// would use, so the password hash is real and login can succeed.
	user, err := authSvc.RegisterWithPolicy(
		context.Background(),
		"alice", "alice@example.com", "hunter2hunter",
		true,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// ----- 1. login --------------------------------------------------------
	loginBody, _ := json.Marshal(map[string]string{
		"username": "alice",
		"password": "hunter2hunter",
	})
	loginResp := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("login: status=%d body=%s", loginResp.Code, loginResp.Body.String())
	}
	if got := loginResp.Header().Get(middleware.HeaderAPIVersion); got == "" {
		t.Fatal("login response missing X-Kite-API-Version header")
	}

	var loginEnv struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken      string    `json:"access_token"`
			RefreshToken     string    `json:"refresh_token"`
			ExpiresAt        time.Time `json:"expires_at"`
			RefreshExpiresAt time.Time `json:"refresh_expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnv); err != nil {
		t.Fatalf("login envelope unparsable: %v body=%s", err, loginResp.Body.String())
	}
	if loginEnv.Code != 0 {
		t.Fatalf("login envelope code=%d, want 0", loginEnv.Code)
	}
	if loginEnv.Data.AccessToken == "" {
		t.Fatal("login response missing access_token")
	}

	access := loginEnv.Data.AccessToken

	// ----- 2. profile via Bearer header -----------------------------------
	profResp := httptest.NewRecorder()
	profReq := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	profReq.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(profResp, profReq)

	if profResp.Code != http.StatusOK {
		t.Fatalf("profile: status=%d body=%s", profResp.Code, profResp.Body.String())
	}
	var profEnv struct {
		Code int `json:"code"`
		Data struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(profResp.Body.Bytes(), &profEnv); err != nil {
		t.Fatalf("profile envelope unparsable: %v", err)
	}
	if profEnv.Data.UserID != user.ID {
		t.Fatalf("profile user_id=%q, want %q", profEnv.Data.UserID, user.ID)
	}
	if profEnv.Data.Username != "alice" {
		t.Fatalf("profile username=%q, want %q", profEnv.Data.Username, "alice")
	}

	// ----- 3. mint PAT, verify it appears in /tokens, then delete it ------
	tokBody, _ := json.Marshal(map[string]any{"name": "test-pat", "expires_in": 7})
	tokResp := httptest.NewRecorder()
	tokReq := httptest.NewRequest(http.MethodPost, "/api/v1/tokens", bytes.NewReader(tokBody))
	tokReq.Header.Set("Content-Type", "application/json")
	tokReq.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(tokResp, tokReq)

	if tokResp.Code != http.StatusCreated {
		t.Fatalf("create token: status=%d body=%s", tokResp.Code, tokResp.Body.String())
	}
	var createdEnv struct {
		Code int `json:"code"`
		Data struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(tokResp.Body.Bytes(), &createdEnv); err != nil {
		t.Fatalf("create token envelope unparsable: %v", err)
	}
	if createdEnv.Data.Token == "" {
		t.Fatal("create token must return plaintext exactly once")
	}
	if createdEnv.Data.Name != "test-pat" {
		t.Fatalf("token name=%q, want %q", createdEnv.Data.Name, "test-pat")
	}

	listResp := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/tokens", nil)
	listReq.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list tokens: status=%d body=%s", listResp.Code, listResp.Body.String())
	}
	if !strings.Contains(listResp.Body.String(), createdEnv.Data.ID) {
		t.Fatalf("created token %q missing from list: %s", createdEnv.Data.ID, listResp.Body.String())
	}
	// And the plaintext token must NOT leak in subsequent reads.
	if strings.Contains(listResp.Body.String(), createdEnv.Data.Token) {
		t.Fatal("plaintext token must NOT appear in subsequent /tokens reads")
	}

	delResp := httptest.NewRecorder()
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/tokens/"+createdEnv.Data.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+access)
	r.ServeHTTP(delResp, delReq)
	if delResp.Code != http.StatusOK {
		t.Fatalf("delete token: status=%d body=%s", delResp.Code, delResp.Body.String())
	}
}

// TestIntegration_LoginBadPassword_EnvelopeShape pins the {code, message,
// data} error envelope on a 401 — this is what every legacy gin client
// already expects. Regressing to huma's default RFC 7807 shape would break
// every client at once.
func TestIntegration_LoginBadPassword_EnvelopeShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, authSvc := newIntegrationServer(t)
	if _, err := authSvc.RegisterWithPolicy(
		context.Background(),
		"bob", "bob@example.com", "rightpassword",
		true,
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"username": "bob", "password": "wrong"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad-password login: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("error envelope unparsable: %v body=%s", err, rec.Body.String())
	}
	if env.Code != 40100 {
		t.Errorf("error code=%d, want 40100 Unauthorized", env.Code)
	}
	if env.Message == "" {
		t.Error("error envelope must carry a non-empty message")
	}
	if string(env.Data) != "null" {
		t.Errorf("error envelope data=%s, want null", env.Data)
	}
}

// TestIntegration_ProfileMissingToken_ReturnsEnvelope nails the
// auth-middleware path: a request without credentials must get the
// envelope-shaped 401, not gin's default text body.
func TestIntegration_ProfileMissingToken_ReturnsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r, _ := newIntegrationServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous profile: status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":40100`) {
		t.Fatalf("expected 40100 envelope, got %s", rec.Body.String())
	}
}

// newIntegrationServer wires a minimal gin engine + auth service + huma
// registration the same way internal/router does in production — the only
// pieces skipped are the unrelated gin handlers (file/album/admin), which
// the typed API doesn't depend on.
func newIntegrationServer(t *testing.T) (*gin.Engine, *service.AuthService) {
	t.Helper()
	id := atomic.AddInt64(&integrationCounter, 1)
	dsn := fmt.Sprintf("file:api-integration-%d?mode=memory&cache=shared", id)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.APIToken{},
		&model.Setting{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	authSvc := service.NewAuthService(
		repo.NewUserRepo(db),
		repo.NewAPITokenRepo(db),
		repo.NewSettingRepo(db),
		config.AuthConfig{
			JWTSecret:          "integration-test-secret",
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			AllowRegistration:  true,
		},
	)

	r := gin.New()
	r.Use(middleware.APIVersion())
	r.Use(GinContextInjector())
	Register(r, Deps{
		AuthSvc: authSvc,
		AuthMW:  middleware.Auth(authSvc),
	})
	return r, authSvc
}
