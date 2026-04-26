package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kite-plus/kite/internal/config"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrUserInactive        = errors.New("user account is inactive")
	ErrUserExists          = errors.New("username or email already exists")
	ErrRegistrationClosed  = errors.New("user registration is not allowed")
	ErrTokenExpired        = errors.New("token has expired")
	ErrTokenInvalid        = errors.New("invalid token")
	ErrPasswordMismatch    = errors.New("current password is incorrect")
	ErrLocalPasswordNotSet = errors.New("local password is not set for this user")
)

// JWTClaims is the claim set carried inside the signed JWT.
//
// PasswordMustChange mirrors the column of the same name on the user row.
// The middleware reads it to block authenticated endpoints until the user
// completes first-login reset — without this claim an attacker who guessed
// the bootstrap admin/admin credentials could skip the reset page and go
// straight to the API, leaving the account forever stuck with the default
// password while still reachable from the outside world. We carry it in
// the claim instead of hitting the DB on every request because the flag
// only matters in a brief window at first login; ResetFirstLoginCredentials
// issues a fresh pair the moment it's cleared.
//
// TokenVersion mirrors the users.token_version counter. The middleware
// rejects any token whose claim is older than the user row's current
// value, which is how credential-changing operations (password change /
// reset / first-login reset / admin-forced reset) invalidate every
// outstanding session issued before them. Without it, a stolen JWT keeps
// working until its natural expiry — the refresh token especially, which
// can live for hours or days depending on config.
type JWTClaims struct {
	UserID             string `json:"user_id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	PasswordMustChange bool   `json:"password_must_change,omitempty"`
	TokenVersion       int    `json:"token_version"`
	jwt.RegisteredClaims
}

// TokenPair bundles an access token with its refresh token. ExpiresAt marks
// when the access token stops validating; RefreshExpiresAt is the longer
// window during which the refresh token can still mint new pairs, and is
// used by the web UI to scope the refresh cookie's MaxAge.
type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// AuthService encapsulates authentication business logic.
type AuthService struct {
	userRepo    *repo.UserRepo
	tokenRepo   *repo.APITokenRepo
	settingRepo *repo.SettingRepo
	cfg         config.AuthConfig
}

func NewAuthService(userRepo *repo.UserRepo, tokenRepo *repo.APITokenRepo, settingRepo *repo.SettingRepo, cfg config.AuthConfig) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		settingRepo: settingRepo,
		cfg:         cfg,
	}
}

// UserRepo exposes the user repository for callers (notably the typed API
// layer in internal/api) that need to read user records without re-deriving
// auth-context plumbing. Returns the same instance NewAuthService received.
func (s *AuthService) UserRepo() *repo.UserRepo { return s.userRepo }

// TokenRepo exposes the API-token repository for the typed API layer.
func (s *AuthService) TokenRepo() *repo.APITokenRepo { return s.tokenRepo }

// Register performs self-service user registration.
func (s *AuthService) Register(ctx context.Context, username, email, password string) (*model.User, error) {
	return s.RegisterWithPolicy(ctx, username, email, password, s.cfg.AllowRegistration)
}

// RegisterWithPolicy performs self-service registration while letting the
// caller provide the effective registration switch (for example a runtime
// setting stored in the database).
func (s *AuthService) RegisterWithPolicy(ctx context.Context, username, email, password string, allowRegistration bool) (*model.User, error) {
	if !allowRegistration {
		return nil, ErrRegistrationClosed
	}

	return s.createUser(ctx, username, email, password, "user", false, true, nil, nil, nil)
}

// CreateStandardUser creates a regular active user account without consulting
// the public self-registration switch. It is intended for administrator flows.
func (s *AuthService) CreateStandardUser(ctx context.Context, username, email, password string) (*model.User, error) {
	return s.CreateStandardUserWithStorageLimit(ctx, username, email, password, nil)
}

// CreateStandardUserWithStorageLimit creates a standard user while allowing
// the caller to override the default storage quota.
func (s *AuthService) CreateStandardUserWithStorageLimit(
	ctx context.Context,
	username, email, password string,
	storageLimit *int64,
) (*model.User, error) {
	return s.createUser(ctx, username, email, password, "user", false, true, nil, nil, storageLimit)
}

// CreateSocialUser creates a new active user that initially only supports
// third-party login. The password hash is set to a random unusable value while
// HasLocalPassword remains false until the user explicitly sets one.
func (s *AuthService) CreateSocialUser(
	ctx context.Context,
	username, email string,
	nickname, avatarURL *string,
) (*model.User, error) {
	return s.CreateSocialUserWithStorageLimit(ctx, username, email, nickname, avatarURL, nil)
}

// CreateSocialUserWithStorageLimit creates a social-login user and optionally
// overrides the default storage quota applied to new regular users.
func (s *AuthService) CreateSocialUserWithStorageLimit(
	ctx context.Context,
	username, email string,
	nickname, avatarURL *string,
	storageLimit *int64,
) (*model.User, error) {
	randomPassword, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate placeholder password: %w", err)
	}
	return s.createUser(ctx, username, email, randomPassword, "user", false, false, nickname, avatarURL, storageLimit)
}

func (s *AuthService) createUser(
	ctx context.Context,
	username, email, password, role string,
	mustChange bool,
	hasLocalPassword bool,
	nickname, avatarURL *string,
	storageLimit *int64,
) (*model.User, error) {
	exists, err := s.userRepo.ExistsByUsernameOrEmail(ctx, username, email)
	if err != nil {
		return nil, fmt.Errorf("register check exists: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("register hash password: %w", err)
	}

	user := &model.User{
		ID:                 uuid.New().String(),
		Username:           username,
		Nickname:           nickname,
		Email:              email,
		AvatarURL:          avatarURL,
		PasswordHash:       string(hash),
		HasLocalPassword:   hasLocalPassword,
		Role:               role,
		StorageLimit:       s.resolveStorageLimit(ctx, role, storageLimit),
		IsActive:           true,
		PasswordMustChange: mustChange,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("register create user: %w", err)
	}
	if !hasLocalPassword {
		user.HasLocalPassword = false
		if err := s.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("persist social password mode: %w", err)
		}
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token pair.
func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenPair, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		// Fall back to email-based lookup.
		user, err = s.userRepo.GetByEmail(ctx, username)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if !user.HasLocalPassword {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokenPair(user)
}

// RefreshToken exchanges a valid refresh token for a new token pair.
//
// The refresh path hits the DB on purpose. Unlike validation on the access
// path — where parse-only is enough because the access token is short
// lived — a refresh token may live for hours or days, and minting a fresh
// pair from a stale one would defeat token_version rotation: a stolen
// refresh token could mint new access tokens indefinitely after a
// password reset. We therefore re-load the user, reject inactive
// accounts, and reject refresh tokens whose TokenVersion claim is older
// than the user row's current value. The new pair embeds the current
// user row's TokenVersion so it continues to validate cleanly.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	if !user.IsActive {
		return nil, ErrTokenInvalid
	}
	if claims.TokenVersion < user.TokenVersion {
		return nil, ErrTokenInvalid
	}

	return s.generateTokenPair(user)
}

// ValidateToken verifies a JWT token and returns its claims.
//
// ValidateToken is parse-only; it does not consult the database, so a
// successful return does not imply the underlying user still exists or
// that the claim's TokenVersion is still current. Callers that care
// about session revocation (password change, account deletion) must
// additionally call [CurrentTokenVersion] and compare.
func (s *AuthService) ValidateToken(tokenStr string) (*JWTClaims, error) {
	return s.parseToken(tokenStr)
}

// CurrentTokenVersion reads the user's current token_version from the
// database. The auth middleware calls it on every authenticated request
// so a credential-changing operation (password change / reset) can
// revoke every outstanding JWT by bumping the counter.
func (s *AuthService) CurrentTokenVersion(ctx context.Context, userID string) (int, error) {
	return s.userRepo.GetTokenVersion(ctx, userID)
}

// ValidateAPIToken verifies an API token and returns the owning user ID.
func (s *AuthService) ValidateAPIToken(ctx context.Context, tokenStr string) (string, error) {
	hash := HashToken(tokenStr)

	token, err := s.tokenRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		return "", ErrTokenInvalid
	}

	if token.IsExpired() {
		return "", ErrTokenExpired
	}

	// Update last-used timestamp asynchronously so the hot path stays fast.
	go func() {
		_ = s.tokenRepo.UpdateLastUsed(context.Background(), token.ID)
	}()

	return token.UserID, nil
}

// CreateAPIToken creates an API token for the user.
// It returns the plaintext token (shown to the user exactly once) alongside the stored record.
func (s *AuthService) CreateAPIToken(ctx context.Context, userID, name string, expiresAt *time.Time) (string, *model.APIToken, error) {
	plainToken, err := generateRandomToken(32)
	if err != nil {
		return "", nil, fmt.Errorf("generate api token: %w", err)
	}

	token := &model.APIToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		TokenHash: HashToken(plainToken),
		ExpiresAt: expiresAt,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return "", nil, fmt.Errorf("create api token: %w", err)
	}

	return plainToken, token, nil
}

// CreateAdminUser creates an administrator account (used by the setup wizard).
// When mustChange is true the user must reset their credentials at first login before anything else.
func (s *AuthService) CreateAdminUser(ctx context.Context, username, email, password string, mustChange bool) (*model.User, error) {
	return s.CreateAdminUserWithStorageLimit(ctx, username, email, password, mustChange, nil)
}

// CreateAdminUserWithStorageLimit creates an administrator account and allows
// callers such as the admin console to override the default unlimited quota.
func (s *AuthService) CreateAdminUserWithStorageLimit(
	ctx context.Context,
	username, email, password string,
	mustChange bool,
	storageLimit *int64,
) (*model.User, error) {
	user, err := s.createUser(ctx, username, email, password, "admin", mustChange, true, nil, nil, storageLimit)
	if err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}
	return user, nil
}

// UpdateProfile updates the mutable profile fields: nickname and avatar.
// Username is immutable (identity anchor); email changes flow through
// EmailChangeService so the user first proves ownership of the new address.
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, newNickname *string, newAvatarURL *string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if newNickname != nil {
		nickname := strings.TrimSpace(*newNickname)
		if nickname == "" {
			user.Nickname = nil
		} else {
			user.Nickname = &nickname
		}
	}
	if newAvatarURL != nil {
		avatar := strings.TrimSpace(*newAvatarURL)
		if avatar == "" {
			user.AvatarURL = nil
		} else {
			user.AvatarURL = &avatar
		}
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

// ChangePassword verifies the current password and writes a new hash.
//
// On success we bump token_version so every outstanding JWT signed
// against the previous value fails the middleware's freshness check.
// This is what turns "password changed" into "existing sessions are
// revoked" — otherwise an attacker who captured a refresh token could
// keep using it after the legitimate user rotated their password.
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if !user.HasLocalPassword {
		return ErrLocalPasswordNotSet
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrPasswordMismatch
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	user.PasswordHash = string(hash)
	user.HasLocalPassword = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if err := s.userRepo.BumpTokenVersion(ctx, userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

// SetPassword sets the user's first local password without requiring a current
// password. Used by accounts created through third-party login.
//
// Bumping token_version here is defensive rather than strictly necessary:
// the user had no local password before this call, so only OAuth-issued
// tokens can exist — but they still need to be invalidated so the
// account cannot be trivially hijacked by someone who stole an OAuth
// session cookie before the password was set.
func (s *AuthService) SetPassword(ctx context.Context, userID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	user.PasswordHash = string(hash)
	user.HasLocalPassword = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if err := s.userRepo.BumpTokenVersion(ctx, userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}
	return nil
}

// ResetFirstLoginCredentials resets the username, email, and password for a first-login user.
// Callable only when PasswordMustChange is true; on success the flag is cleared and a fresh
// token pair is issued because a username change invalidates the username claim on the old token.
func (s *AuthService) ResetFirstLoginCredentials(ctx context.Context, userID, newUsername, newEmail, newPassword string) (*TokenPair, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if !user.PasswordMustChange {
		return nil, errors.New("first-login reset is not required for this user")
	}

	// Username/email conflict check, excluding the user themself.
	if newUsername != user.Username || newEmail != user.Email {
		conflict, err := s.userRepo.ExistsByUsernameOrEmailExcept(ctx, newUsername, newEmail, userID)
		if err != nil {
			return nil, fmt.Errorf("check conflict: %w", err)
		}
		if conflict {
			return nil, ErrUserExists
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash new password: %w", err)
	}

	user.Username = newUsername
	user.Email = newEmail
	user.PasswordHash = string(hash)
	user.HasLocalPassword = true
	user.PasswordMustChange = false

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	// Revoke any JWT minted off the bootstrap credentials. The fresh
	// pair we return below is re-read from the now-bumped row so its
	// TokenVersion claim still validates.
	if err := s.userRepo.BumpTokenVersion(ctx, userID); err != nil {
		return nil, fmt.Errorf("revoke sessions: %w", err)
	}
	user.TokenVersion++

	return s.generateTokenPair(user)
}

func (s *AuthService) resolveStorageLimit(ctx context.Context, role string, explicit *int64) int64 {
	if explicit != nil {
		return *explicit
	}
	if role == "admin" {
		return -1
	}
	return s.defaultStandardUserStorageLimit(ctx)
}

func (s *AuthService) defaultStandardUserStorageLimit(ctx context.Context) int64 {
	if s.settingRepo == nil {
		return DefaultStorageQuotaBytes()
	}

	raw, err := s.settingRepo.GetOrDefault(ctx, DefaultQuotaSettingKey, DefaultQuotaSettingValue())
	if err != nil {
		return DefaultStorageQuotaBytes()
	}

	limit, err := ParseStorageQuotaBytes(raw)
	if err != nil {
		return DefaultStorageQuotaBytes()
	}

	return limit
}

// IssueTokenPair issues a fresh token pair for the given user without
// re-authenticating a password flow. Used after third-party login succeeds.
func (s *AuthService) IssueTokenPair(user *model.User) (*TokenPair, error) {
	return s.generateTokenPair(user)
}

func (s *AuthService) generateTokenPair(user *model.User) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.cfg.AccessTokenExpiry)
	refreshExpiry := now.Add(s.cfg.RefreshTokenExpiry)

	accessClaims := &JWTClaims{
		UserID:             user.ID,
		Username:           user.Username,
		Role:               user.Role,
		PasswordMustChange: user.PasswordMustChange,
		TokenVersion:       user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := &JWTClaims{
		UserID:             user.ID,
		Username:           user.Username,
		Role:               user.Role,
		PasswordMustChange: user.PasswordMustChange,
		TokenVersion:       user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresAt:        accessExpiry,
		RefreshExpiresAt: refreshExpiry,
	}, nil
}

func (s *AuthService) parseToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// HashToken returns the SHA256 hex digest of token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
