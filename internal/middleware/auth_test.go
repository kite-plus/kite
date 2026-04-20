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
	if err := db.AutoMigrate(&model.User{}, &model.APIToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM users")
		db.Exec("DELETE FROM api_tokens")
	})
	return db
}

func newAuthSvc(db *gorm.DB) *service.AuthService {
	return service.NewAuthService(
		repo.NewUserRepo(db),
		repo.NewAPITokenRepo(db),
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
