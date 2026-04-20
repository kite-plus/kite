package service

import (
	"context"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newAuthTestDB(t *testing.T) *gorm.DB {
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

func newAuthService(db *gorm.DB, allowReg bool) *AuthService {
	return NewAuthService(
		repo.NewUserRepo(db),
		repo.NewAPITokenRepo(db),
		config.AuthConfig{
			JWTSecret:          "test-secret",
			AccessTokenExpiry:  time.Hour,
			RefreshTokenExpiry: 24 * time.Hour,
			AllowRegistration:  allowReg,
		},
	)
}

func TestAuthService_Register(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	user, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.ID == "" || user.Role != "user" || !user.IsActive {
		t.Fatalf("unexpected user: %+v", user)
	}
	if user.PasswordHash == "password123" {
		t.Fatal("password should be hashed")
	}

	if _, err := svc.Register(context.Background(), "alice", "other@example.com", "x"); err != ErrUserExists {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestAuthService_Register_ClosedRegistration(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, false)

	_, err := svc.Register(context.Background(), "bob", "bob@example.com", "pw")
	if err != ErrRegistrationClosed {
		t.Fatalf("expected ErrRegistrationClosed, got %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	if _, err := svc.Register(context.Background(), "carol", "carol@example.com", "hunter2"); err != nil {
		t.Fatalf("register: %v", err)
	}

	pair, err := svc.Login(context.Background(), "carol", "hunter2")
	if err != nil {
		t.Fatalf("login by username: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("token pair empty")
	}

	if _, err := svc.Login(context.Background(), "carol@example.com", "hunter2"); err != nil {
		t.Fatalf("login by email: %v", err)
	}

	if _, err := svc.Login(context.Background(), "carol", "wrong"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := svc.Login(context.Background(), "ghost", "x"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for missing user, got %v", err)
	}
}

func TestAuthService_Login_InactiveUser(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	user, err := svc.Register(context.Background(), "dave", "dave@example.com", "pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := repo.NewUserRepo(db).Delete(context.Background(), user.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	if _, err := svc.Login(context.Background(), "dave", "pw"); err != ErrUserInactive {
		t.Fatalf("expected ErrUserInactive, got %v", err)
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	if _, err := svc.Register(context.Background(), "eve", "eve@example.com", "pw"); err != nil {
		t.Fatalf("register: %v", err)
	}
	pair, err := svc.Login(context.Background(), "eve", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	claims, err := svc.ValidateToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.Username != "eve" || claims.Role != "user" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	if _, err := svc.ValidateToken("garbage"); err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}

	wrongSvc := NewAuthService(
		repo.NewUserRepo(db),
		repo.NewAPITokenRepo(db),
		config.AuthConfig{JWTSecret: "different", AccessTokenExpiry: time.Hour, RefreshTokenExpiry: time.Hour},
	)
	if _, err := wrongSvc.ValidateToken(pair.AccessToken); err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid for wrong secret, got %v", err)
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	if _, err := svc.Register(context.Background(), "frank", "frank@example.com", "pw"); err != nil {
		t.Fatalf("register: %v", err)
	}
	pair, err := svc.Login(context.Background(), "frank", "pw")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	next, err := svc.RefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if next.AccessToken == "" {
		t.Fatal("empty access token after refresh")
	}

	if _, err := svc.RefreshToken("not-a-token"); err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestAuthService_CreateAndValidateAPIToken(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	user, err := svc.Register(context.Background(), "grace", "grace@example.com", "pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	plain, stored, err := svc.CreateAPIToken(context.Background(), user.ID, "picgo", nil)
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}
	if plain == "" || stored.ID == "" {
		t.Fatalf("unexpected plain/stored: %q / %+v", plain, stored)
	}
	if stored.TokenHash == plain {
		t.Fatal("stored hash must differ from plaintext")
	}
	if stored.TokenHash != HashToken(plain) {
		t.Fatal("stored hash does not match HashToken(plain)")
	}

	userID, err := svc.ValidateAPIToken(context.Background(), plain)
	if err != nil {
		t.Fatalf("validate api token: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("unexpected userID: %s want %s", userID, user.ID)
	}

	if _, err := svc.ValidateAPIToken(context.Background(), "wrong-token"); err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestAuthService_ValidateAPIToken_Expired(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	user, err := svc.Register(context.Background(), "hank", "hank@example.com", "pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	plain, _, err := svc.CreateAPIToken(context.Background(), user.ID, "expired", &past)
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}

	if _, err := svc.ValidateAPIToken(context.Background(), plain); err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestAuthService_CreateAdminUser(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	admin, err := svc.CreateAdminUser(context.Background(), "root", "root@example.com", "secret", true)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if admin.Role != "admin" || admin.StorageLimit != -1 || !admin.PasswordMustChange {
		t.Fatalf("unexpected admin: %+v", admin)
	}
}

func TestAuthService_ChangePassword(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	user, err := svc.Register(context.Background(), "ivy", "ivy@example.com", "old-pw")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.ChangePassword(context.Background(), user.ID, "wrong", "new"); err != ErrPasswordMismatch {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}

	if err := svc.ChangePassword(context.Background(), user.ID, "old-pw", "new-pw"); err != nil {
		t.Fatalf("change password: %v", err)
	}

	if _, err := svc.Login(context.Background(), "ivy", "new-pw"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestAuthService_UpdateProfile(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	u1, _ := svc.Register(context.Background(), "jack", "jack@example.com", "pw")
	_, _ = svc.Register(context.Background(), "jill", "jill@example.com", "pw")

	nickname := "Captain"
	avatar := "https://example.com/a.png"
	updated, err := svc.UpdateProfile(context.Background(), u1.ID, "jack2", &nickname, "jack2@example.com", &avatar)
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Username != "jack2" || updated.Email != "jack2@example.com" {
		t.Fatalf("unexpected update: %+v", updated)
	}
	if updated.Nickname == nil || *updated.Nickname != "Captain" {
		t.Fatalf("unexpected nickname: %v", updated.Nickname)
	}

	if _, err := svc.UpdateProfile(context.Background(), u1.ID, "jill", nil, "jack2@example.com", nil); err != ErrUserExists {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}

	empty := ""
	cleared, err := svc.UpdateProfile(context.Background(), u1.ID, "jack2", &empty, "jack2@example.com", &empty)
	if err != nil {
		t.Fatalf("clear profile fields: %v", err)
	}
	if cleared.Nickname != nil || cleared.AvatarURL != nil {
		t.Fatalf("expected nickname/avatar cleared: %+v", cleared)
	}
}

func TestAuthService_ResetFirstLoginCredentials(t *testing.T) {
	db := newAuthTestDB(t)
	svc := newAuthService(db, true)

	admin, err := svc.CreateAdminUser(context.Background(), "setup", "setup@example.com", "temp", true)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	pair, err := svc.ResetFirstLoginCredentials(context.Background(), admin.ID, "admin", "admin@example.com", "new-pw")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("empty access token after reset")
	}

	// Reset should not be callable twice.
	if _, err := svc.ResetFirstLoginCredentials(context.Background(), admin.ID, "admin", "admin@example.com", "x"); err == nil {
		t.Fatal("expected error on second reset")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	a := HashToken("abc")
	b := HashToken("abc")
	if a != b {
		t.Fatalf("HashToken not deterministic: %s vs %s", a, b)
	}
	if HashToken("abc") == HashToken("abd") {
		t.Fatal("HashToken collision on different inputs")
	}
	// SHA256 hex length is 64.
	if len(a) != 64 {
		t.Fatalf("HashToken length %d want 64", len(a))
	}
}
