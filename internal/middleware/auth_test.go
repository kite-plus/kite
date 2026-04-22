package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newMiddlewareTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.APIToken{}, &model.Setting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM users")
		db.Exec("DELETE FROM api_tokens")
		db.Exec("DELETE FROM settings")
	})
	return db
}

func newAuthSvc(db *gorm.DB) *service.AuthService {
	return service.NewAuthService(
		repo.NewUserRepo(db),
		repo.NewAPITokenRepo(db),
		repo.NewSettingRepo(db),
		config.AuthConfig{
			JWTSecret:          "middleware-secret",
			AccessTokenExpiry:  time.Hour,
			RefreshTokenExpiry: 2 * time.Hour,
			AllowRegistration:  true,
		},
	)
}

func setupAuthRouter(authSvc *service.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/secured", Auth(authSvc), func(c *gin.Context) {
		uid, _ := c.Get(ContextKeyUserID)
		role, _ := c.Get(ContextKeyRole)
		c.JSON(http.StatusOK, gin.H{"uid": uid, "role": role})
	})
	r.GET("/admin-only", Auth(authSvc), AdminOnly(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestAuth_MissingToken(t *testing.T) {
	db := newMiddlewareTestDB(t)
	r := setupAuthRouter(newAuthSvc(db))

	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", w.Code)
	}
}

func TestAuth_ValidJWTHeader(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	user, err := svc.Register(context.Background(), "m1", "m1@example.com", "pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	pair, err := svc.Login(context.Background(), "m1", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	r := setupAuthRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if user.ID == "" {
		t.Fatal("user ID unexpectedly empty")
	}
}

func TestAuth_ValidJWTCookie(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	if _, err := svc.Register(context.Background(), "m2", "m2@example.com", "pw"); err != nil {
		t.Fatalf("register: %v", err)
	}
	pair, err := svc.Login(context.Background(), "m2", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	r := setupAuthRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: pair.AccessToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	db := newMiddlewareTestDB(t)
	r := setupAuthRouter(newAuthSvc(db))

	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.Header.Set("Authorization", "Bearer garbage-value")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", w.Code)
	}
}

func TestAuth_FallsBackToAPIToken(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	user, err := svc.Register(context.Background(), "m3", "m3@example.com", "pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	plain, _, err := svc.CreateAPIToken(context.Background(), user.ID, "picgo", nil)
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}

	r := setupAuthRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminOnly_RejectsNonAdmin(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	if _, err := svc.Register(context.Background(), "m4", "m4@example.com", "pw"); err != nil {
		t.Fatalf("register: %v", err)
	}
	pair, err := svc.Login(context.Background(), "m4", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	r := setupAuthRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", w.Code)
	}
}

// TestAuth_FirstLoginGate_BlocksNonAllowedRoute verifies that a user whose
// JWT carries PasswordMustChange=true cannot call endpoints outside the
// whitelist — without this backend gate, anyone holding the default
// admin/admin bootstrap credentials could skip the reset UI by hitting the
// API directly and use the account indefinitely.
func TestAuth_FirstLoginGate_BlocksNonAllowedRoute(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	// CreateAdminUser with mustChange=true mirrors the bootstrap seeding
	// path in cmd/kite/main.go that creates the default admin account.
	if _, err := svc.CreateAdminUser(context.Background(), "boot", "boot@example.com", "admin", true); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	pair, err := svc.Login(context.Background(), "boot", "admin")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	r := setupAuthRouter(svc)
	// /secured is a generic authed route, not on the first-login whitelist.
	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403 (first-login gate)", w.Code)
	}
}

// TestAuth_FirstLoginGate_AllowsWhitelistedRoutes verifies that the three
// endpoints a first-login user must be able to hit — /profile,
// /auth/first-login-reset, /auth/logout — pass through. The test registers
// matching stub routes because the middleware keys off the gin route
// pattern rather than the URL string.
func TestAuth_FirstLoginGate_AllowsWhitelistedRoutes(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	if _, err := svc.CreateAdminUser(context.Background(), "boot2", "boot2@example.com", "admin", true); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	pair, err := svc.Login(context.Background(), "boot2", "admin")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/api/v1/profile", Auth(svc), ok)
	r.POST("/api/v1/auth/first-login-reset", Auth(svc), ok)
	r.POST("/api/v1/auth/logout", Auth(svc), ok)

	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"profile", http.MethodGet, "/api/v1/profile"},
		{"first-login-reset", http.MethodPost, "/api/v1/auth/first-login-reset"},
		{"logout", http.MethodPost, "/api/v1/auth/logout"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("%s: code=%d want 200; body=%s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestAuth_FirstLoginGate_PassesAfterReset verifies that once
// ResetFirstLoginCredentials clears the flag, the freshly issued token
// pair no longer trips the first-login gate.
func TestAuth_FirstLoginGate_PassesAfterReset(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	admin, err := svc.CreateAdminUser(context.Background(), "boot3", "boot3@example.com", "admin", true)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	// Reset the credentials, which clears PasswordMustChange and returns a
	// fresh pair carrying the updated claim.
	pair, err := svc.ResetFirstLoginCredentials(context.Background(), admin.ID, "postboot", "postboot@example.com", "brand-new-pw")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}

	r := setupAuthRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/secured", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 after reset; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminOnly_AllowsAdmin(t *testing.T) {
	db := newMiddlewareTestDB(t)
	svc := newAuthSvc(db)
	admin, err := svc.CreateAdminUser(context.Background(), "boss", "boss@example.com", "pw", false)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	_ = admin
	pair, err := svc.Login(context.Background(), "boss", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	r := setupAuthRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", w.Code)
	}
}
