package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var socialAuthDBCounter int64

func newSocialAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&socialAuthDBCounter, 1)
	dsn := fmt.Sprintf("file:social-auth-test-%d?mode=memory&cache=shared", id)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserIdentity{}, &model.APIToken{}, &model.Setting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newSocialAuthTestServices(t *testing.T, allowRegistration bool) (*AuthService, *SocialAuthService, *repo.UserRepo, *repo.UserIdentityRepo, *repo.SettingRepo) {
	t.Helper()
	db := newSocialAuthTestDB(t)
	userRepo := repo.NewUserRepo(db)
	identityRepo := repo.NewUserIdentityRepo(db)
	settingRepo := repo.NewSettingRepo(db)
	authSvc := NewAuthService(
		userRepo,
		repo.NewAPITokenRepo(db),
		config.AuthConfig{
			JWTSecret:          "test-secret",
			AccessTokenExpiry:  time.Hour,
			RefreshTokenExpiry: 24 * time.Hour,
			AllowRegistration:  allowRegistration,
		},
	)
	oauthConfigSvc := NewOAuthConfigService(settingRepo, "https://kite.test")
	socialSvc := NewSocialAuthService(authSvc, userRepo, identityRepo, settingRepo, oauthConfigSvc, "test-secret", allowRegistration)
	return authSvc, socialSvc, userRepo, identityRepo, settingRepo
}

func TestOAuthConfigService_ListPublicProviders_RequiresCompleteConfig(t *testing.T) {
	_, _, _, _, settingRepo := newSocialAuthTestServices(t, true)
	ctx := context.Background()
	if err := settingRepo.SetBatch(ctx, map[string]string{
		"site_url":                   "https://kite.test",
		"oauth_github_enabled":       "true",
		"oauth_github_client_id":     "github-id",
		"oauth_github_client_secret": "github-secret",
		"oauth_google_enabled":       "true",
		"oauth_google_client_id":     "google-id",
		"oauth_google_client_secret": "",
		"oauth_wechat_enabled":       "true",
		"oauth_wechat_client_id":     "wx-id",
		"oauth_wechat_client_secret": "wx-secret",
	}); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	svc := NewOAuthConfigService(settingRepo, "https://kite.test")
	providers, err := svc.ListPublicProviders(ctx)
	if err != nil {
		t.Fatalf("list public providers: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("expected 2 public providers, got %d", len(providers))
	}
	if providers[0].Key != "wechat" || providers[1].Key != "github" {
		t.Fatalf("unexpected provider order: %+v", providers)
	}
}

func TestSocialAuthService_ResolveLoginRedirect_MergesVerifiedEmail(t *testing.T) {
	authSvc, socialSvc, userRepo, identityRepo, _ := newSocialAuthTestServices(t, true)
	ctx := context.Background()
	user, err := authSvc.CreateStandardUser(ctx, "alice", "alice@example.com", "hunter2")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	redirectTo, err := socialSvc.resolveLoginRedirect(ctx, "/user/dashboard", &SocialProfile{
		Provider:       "github",
		ProviderUserID: "github-1",
		Email:          stringPtr("alice@example.com"),
		EmailVerified:  true,
		DisplayName:    stringPtr("Alice"),
	})
	if err != nil {
		t.Fatalf("resolve login redirect: %v", err)
	}
	if !strings.HasPrefix(redirectTo, "/login/callback?ticket=") {
		t.Fatalf("unexpected redirect: %s", redirectTo)
	}

	identity, err := identityRepo.GetByProviderUserID(ctx, "github", "github-1")
	if err != nil {
		t.Fatalf("get identity: %v", err)
	}
	if identity.UserID != user.ID {
		t.Fatalf("expected identity to bind user %s, got %s", user.ID, identity.UserID)
	}

	savedUser, err := userRepo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get merged user: %v", err)
	}
	if savedUser.Email != "alice@example.com" {
		t.Fatalf("unexpected merged user: %+v", savedUser)
	}
}

func TestSocialAuthService_CompleteOnboarding_CreatesSocialUser(t *testing.T) {
	_, socialSvc, userRepo, identityRepo, _ := newSocialAuthTestServices(t, true)
	ctx := context.Background()
	redirectTo, err := socialSvc.resolveLoginRedirect(ctx, "/user/dashboard", &SocialProfile{
		Provider:       "wechat",
		ProviderUserID: "wx-openid-1",
		DisplayName:    stringPtr("微信用户"),
	})
	if err != nil {
		t.Fatalf("resolve onboarding redirect: %v", err)
	}

	parsed, err := url.Parse(redirectTo)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	ticket := parsed.Query().Get("ticket")
	if ticket == "" {
		t.Fatalf("expected onboarding ticket in redirect: %s", redirectTo)
	}

	pair, user, returnTo, err := socialSvc.CompleteOnboarding(ctx, ticket, "wechat-user", "wechat@example.com")
	if err != nil {
		t.Fatalf("complete onboarding: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("expected token pair, got %+v", pair)
	}
	if returnTo != "/user/dashboard" {
		t.Fatalf("unexpected returnTo: %s", returnTo)
	}

	savedUser, err := userRepo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get created user: %v", err)
	}
	if savedUser.HasLocalPassword {
		t.Fatalf("expected social-only user without local password, got %+v", savedUser)
	}

	identity, err := identityRepo.GetByProviderUserID(ctx, "wechat", "wx-openid-1")
	if err != nil {
		t.Fatalf("get created identity: %v", err)
	}
	if identity.UserID != user.ID {
		t.Fatalf("expected identity to bind created user %s, got %s", user.ID, identity.UserID)
	}
}

func TestSocialAuthService_UnlinkIdentity_BlocksLastLoginMethod(t *testing.T) {
	authSvc, socialSvc, _, _, _ := newSocialAuthTestServices(t, true)
	ctx := context.Background()
	user, err := authSvc.CreateSocialUser(ctx, "wechat-user", "wechat@example.com", stringPtr("微信用户"), nil)
	if err != nil {
		t.Fatalf("create social user: %v", err)
	}
	if _, err := socialSvc.linkIdentityToUser(ctx, user.ID, &SocialProfile{
		Provider:       "wechat",
		ProviderUserID: "wx-openid-1",
		DisplayName:    stringPtr("微信用户"),
	}); err != nil {
		t.Fatalf("link identity: %v", err)
	}

	err = socialSvc.UnlinkIdentity(ctx, user.ID, "wechat")
	if !errors.Is(err, ErrOAuthLastLoginMethod) {
		t.Fatalf("expected ErrOAuthLastLoginMethod, got %v", err)
	}
}
