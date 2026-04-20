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

	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair bundles an access token with its refresh token.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthService encapsulates authentication business logic.
type AuthService struct {
	userRepo  *repo.UserRepo
	tokenRepo *repo.APITokenRepo
	cfg       config.AuthConfig
}

func NewAuthService(userRepo *repo.UserRepo, tokenRepo *repo.APITokenRepo, cfg config.AuthConfig) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}
}

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

	return s.createUser(ctx, username, email, password, "user", false, true, nil, nil)
}

// CreateStandardUser creates a regular active user account without consulting
// the public self-registration switch. It is intended for administrator flows.
func (s *AuthService) CreateStandardUser(ctx context.Context, username, email, password string) (*model.User, error) {
	return s.createUser(ctx, username, email, password, "user", false, true, nil, nil)
}

// CreateSocialUser creates a new active user that initially only supports
// third-party login. The password hash is set to a random unusable value while
// HasLocalPassword remains false until the user explicitly sets one.
func (s *AuthService) CreateSocialUser(
	ctx context.Context,
	username, email string,
	nickname, avatarURL *string,
) (*model.User, error) {
	randomPassword, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate placeholder password: %w", err)
	}
	return s.createUser(ctx, username, email, randomPassword, "user", false, false, nickname, avatarURL)
}

func (s *AuthService) createUser(
	ctx context.Context,
	username, email, password, role string,
	mustChange bool,
	hasLocalPassword bool,
	nickname, avatarURL *string,
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
func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}

	return s.generateTokenPair(user)
}

// ValidateToken verifies a JWT token and returns its claims.
func (s *AuthService) ValidateToken(tokenStr string) (*JWTClaims, error) {
	return s.parseToken(tokenStr)
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
	user, err := s.createUser(ctx, username, email, password, "admin", mustChange, true, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}
	user.StorageLimit = -1 // administrators have no storage quota
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("persist admin storage limit: %w", err)
	}

	return user, nil
}

// UpdateProfile updates the current user's profile fields (username, nickname, email, avatar).
// Passwords are unchanged; username and email changes are checked for uniqueness.
func (s *AuthService) UpdateProfile(ctx context.Context, userID, newUsername string, newNickname *string, newEmail string, newAvatarURL *string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if newUsername != user.Username || newEmail != user.Email {
		conflict, err := s.userRepo.ExistsByUsernameOrEmailExcept(ctx, newUsername, newEmail, userID)
		if err != nil {
			return nil, fmt.Errorf("check conflict: %w", err)
		}
		if conflict {
			return nil, ErrUserExists
		}
	}

	user.Username = newUsername
	if newNickname != nil {
		nickname := strings.TrimSpace(*newNickname)
		if nickname == "" {
			user.Nickname = nil
		} else {
			user.Nickname = &nickname
		}
	}
	user.Email = newEmail
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
	return nil
}

// SetPassword sets the user's first local password without requiring a current
// password. Used by accounts created through third-party login.
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

	return s.generateTokenPair(user)
}

// IssueTokenPair issues a fresh token pair for the given user without
// re-authenticating a password flow. Used after third-party login succeeds.
func (s *AuthService) IssueTokenPair(user *model.User) (*TokenPair, error) {
	return s.generateTokenPair(user)
}

func (s *AuthService) generateTokenPair(user *model.User) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.cfg.AccessTokenExpiry)

	accessClaims := &JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
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
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiry,
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
