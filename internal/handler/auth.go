package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/middleware"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication HTTP requests.
type AuthHandler struct {
	authSvc                  *service.AuthService
	socialAuthSvc            *service.SocialAuthService
	oauthConfigSvc           *service.OAuthConfigService
	emailChangeSvc           *service.EmailChangeService
	userRepo                 *repo.UserRepo
	settingRepo              *repo.SettingRepo
	allowRegistrationDefault bool
	uploadMaxFileSizeDefault int64
}

func NewAuthHandler(
	authSvc *service.AuthService,
	socialAuthSvc *service.SocialAuthService,
	oauthConfigSvc *service.OAuthConfigService,
	emailChangeSvc *service.EmailChangeService,
	userRepo *repo.UserRepo,
	settingRepo *repo.SettingRepo,
	allowRegistrationDefault bool,
	uploadMaxFileSizeDefault int64,
) *AuthHandler {
	return &AuthHandler{
		authSvc:                  authSvc,
		socialAuthSvc:            socialAuthSvc,
		oauthConfigSvc:           oauthConfigSvc,
		emailChangeSvc:           emailChangeSvc,
		userRepo:                 userRepo,
		settingRepo:              settingRepo,
		allowRegistrationDefault: allowRegistrationDefault,
		uploadMaxFileSizeDefault: uploadMaxFileSizeDefault,
	}
}

func (h *AuthHandler) allowRegistration(c *gin.Context) bool {
	if h.settingRepo == nil {
		return h.allowRegistrationDefault
	}
	enabled, err := h.settingRepo.GetBool(c.Request.Context(), "allow_registration", h.allowRegistrationDefault)
	if err != nil {
		return h.allowRegistrationDefault
	}
	return enabled
}

// Options returns the effective public auth settings for anonymous pages.
func (h *AuthHandler) Options(c *gin.Context) {
	socialProviders := make([]service.PublicOAuthProvider, 0)
	if h.oauthConfigSvc != nil {
		providers, err := h.oauthConfigSvc.ListPublicProviders(c.Request.Context())
		if err == nil {
			socialProviders = providers
		}
	}
	uploadMaxFileSizeMB := service.DefaultUploadMaxFileSizeMB(h.uploadMaxFileSizeDefault)
	uploadMaxFileSizeBytes := h.uploadMaxFileSizeDefault
	if h.settingRepo != nil {
		if saved, err := h.settingRepo.GetOrDefault(c.Request.Context(), service.UploadMaxFileSizeMBSettingKey, uploadMaxFileSizeMB); err == nil {
			uploadMaxFileSizeMB = saved
			if parsed, parseErr := service.ParseUploadMaxFileSizeBytes(saved); parseErr == nil {
				uploadMaxFileSizeBytes = parsed
			}
		}
	}

	Success(c, gin.H{
		"allow_registration":         h.allowRegistration(c),
		"social_providers":           socialProviders,
		"upload_max_file_size_mb":    uploadMaxFileSizeMB,
		"upload_max_file_size_bytes": uploadMaxFileSizeBytes,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user and returns a token pair.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "username and password are required")
		return
	}

	tokenPair, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserInactive) {
			Unauthorized(c, err.Error())
			return
		}
		ServerError(c, "login failed")
		return
	}

	// The web UI relies exclusively on HttpOnly cookies — the JSON body keeps
	// non-browser clients (CLIs, mobile, integration tests) working.
	writeAuthCookies(c, tokenPair)

	Success(c, tokenPair)
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

// Register creates a new user account via self-registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid registration data: "+err.Error())
		return
	}

	user, err := h.authSvc.RegisterWithPolicy(
		c.Request.Context(),
		req.Username,
		req.Email,
		req.Password,
		h.allowRegistration(c),
	)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationClosed) {
			Forbidden(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserExists) {
			Fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
		ServerError(c, "registration failed")
		return
	}

	Created(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken exchanges a refresh token for a new access token. The token is
// read from the HttpOnly cookie for browser clients, with a fallback to the
// JSON body for API/CLI clients that don't use cookies.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	token := extractRefreshToken(c)
	if token == "" {
		Unauthorized(c, "refresh_token is required")
		return
	}

	tokenPair, err := h.authSvc.RefreshToken(c.Request.Context(), token)
	if err != nil {
		Unauthorized(c, "invalid refresh token")
		return
	}

	// Refresh rotation — issue a fresh cookie pair so the refresh token itself
	// is replaced on every use. The body still carries the new pair for CLI
	// clients.
	writeAuthCookies(c, tokenPair)
	Success(c, tokenPair)
}

// Logout clears the access and refresh cookies. The access token in the
// Authorization header is still usable until it expires — there's no
// server-side revocation store — but the browser session is gone.
func (h *AuthHandler) Logout(c *gin.Context) {
	writeAccessTokenCookie(c, "", time.Unix(0, 0))
	writeRefreshTokenCookie(c, "", time.Unix(0, 0))
	Success(c, nil)
}

// extractRefreshToken returns the refresh token from (in order) the HttpOnly
// cookie and the JSON body. The body path exists so integration tests and
// non-browser API consumers can still call /auth/refresh.
func extractRefreshToken(c *gin.Context) string {
	if cookie, err := c.Cookie(refreshTokenCookieName); err == nil {
		if t := strings.TrimSpace(cookie); t != "" {
			return t
		}
	}
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err == nil {
		return strings.TrimSpace(req.RefreshToken)
	}
	return ""
}

// GetProfile returns information about the currently authenticated user.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		Unauthorized(c, "user not found")
		return
	}

	Success(c, gin.H{
		"user_id":              user.ID,
		"username":             user.Username,
		"nickname":             user.Nickname,
		"email":                user.Email,
		"avatar_url":           user.AvatarURL,
		"has_local_password":   user.HasLocalPassword,
		"role":                 user.Role,
		"password_must_change": user.PasswordMustChange,
		"storage_limit":        user.StorageLimit,
		"storage_used":         user.StorageUsed,
		"created_at":           user.CreatedAt,
	})
}

type updateProfileRequest struct {
	Nickname  *string `json:"nickname" binding:"omitempty,max=32"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,max=512"`
}

// UpdateProfile lets the current user update their mutable profile fields
// (nickname and avatar). Username is immutable; email changes go through
// the verified flow exposed at /auth/email-change/*.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid profile data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.authSvc.UpdateProfile(c.Request.Context(), userID, req.Nickname, req.AvatarURL)
	if err != nil {
		ServerError(c, "update profile failed")
		return
	}

	Success(c, gin.H{
		"user_id":            user.ID,
		"username":           user.Username,
		"nickname":           user.Nickname,
		"email":              user.Email,
		"avatar_url":         user.AvatarURL,
		"has_local_password": user.HasLocalPassword,
		"role":               user.Role,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=64"`
}

// ChangePassword lets the current user change their own password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid password data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrLocalPasswordNotSet) {
			Fail(c, http.StatusBadRequest, 40011, err.Error())
			return
		}
		if errors.Is(err, service.ErrPasswordMismatch) {
			Fail(c, http.StatusBadRequest, 40010, err.Error())
			return
		}
		ServerError(c, "change password failed")
		return
	}

	Success(c, nil)
}

type firstLoginResetRequest struct {
	NewUsername string `json:"new_username" binding:"required,min=3,max=32"`
	NewEmail    string `json:"new_email" binding:"required,email"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

// FirstLoginReset forces a first-login account and password reset.
// Allowed only when PasswordMustChange=true; returns a fresh token pair on success.
func (h *AuthHandler) FirstLoginReset(c *gin.Context) {
	var req firstLoginResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid reset data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	tokenPair, err := h.authSvc.ResetFirstLoginCredentials(
		c.Request.Context(), userID, req.NewUsername, req.NewEmail, req.NewPassword,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			Fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
		BadRequest(c, err.Error())
		return
	}

	writeAuthCookies(c, tokenPair)
	Success(c, tokenPair)
}

type requestEmailChangeRequest struct {
	NewEmail string `json:"new_email" binding:"required,email,max=254"`
}

// RequestEmailChange sends a verification code to the requested new email so
// the caller can prove ownership before the address is rotated on the account.
func (h *AuthHandler) RequestEmailChange(c *gin.Context) {
	if h.emailChangeSvc == nil {
		ServerError(c, "email change service unavailable")
		return
	}

	var req requestEmailChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid email: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	expiresAt, err := h.emailChangeSvc.RequestEmailChange(c.Request.Context(), userID, req.NewEmail)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmailFormat):
			Fail(c, http.StatusBadRequest, 40001, "invalid email address")
		case errors.Is(err, service.ErrSameAsCurrentEmail):
			Fail(c, http.StatusBadRequest, 40002, "new email matches current email")
		case errors.Is(err, service.ErrEmailTaken):
			Fail(c, http.StatusConflict, 40900, "email is already registered")
		case errors.Is(err, service.ErrVerificationCooldown):
			Fail(c, http.StatusTooManyRequests, 42900, "please wait before requesting another code")
		case errors.Is(err, service.ErrSMTPNotConfigured):
			Fail(c, http.StatusFailedDependency, 42400, err.Error())
		default:
			ServerError(c, "send verification email failed: "+err.Error())
		}
		return
	}

	Success(c, gin.H{
		"expires_at":     expiresAt,
		"resend_after_s": 60,
	})
}

type confirmEmailChangeRequest struct {
	NewEmail string `json:"new_email" binding:"required,email,max=254"`
	Code     string `json:"code" binding:"required,min=4,max=12"`
}

// ConfirmEmailChange verifies the code and, on success, rotates the user's
// email address. Returns the refreshed profile so the client can update its
// cache without a second round trip.
func (h *AuthHandler) ConfirmEmailChange(c *gin.Context) {
	if h.emailChangeSvc == nil {
		ServerError(c, "email change service unavailable")
		return
	}

	var req confirmEmailChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid payload: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.emailChangeSvc.ConfirmEmailChange(c.Request.Context(), userID, req.NewEmail, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmailFormat):
			Fail(c, http.StatusBadRequest, 40001, "invalid email address")
		case errors.Is(err, service.ErrVerificationNotFound):
			Fail(c, http.StatusNotFound, 40400, "no pending verification for this email")
		case errors.Is(err, service.ErrVerificationCodeWrong):
			Fail(c, http.StatusBadRequest, 40003, "verification code is incorrect")
		case errors.Is(err, service.ErrVerificationExpired):
			Fail(c, http.StatusGone, 41000, "verification code has expired")
		case errors.Is(err, service.ErrEmailTaken):
			Fail(c, http.StatusConflict, 40900, "email is already registered")
		default:
			ServerError(c, "confirm email change failed: "+err.Error())
		}
		return
	}

	Success(c, gin.H{
		"user_id":            user.ID,
		"username":           user.Username,
		"nickname":           user.Nickname,
		"email":              user.Email,
		"avatar_url":         user.AvatarURL,
		"has_local_password": user.HasLocalPassword,
		"role":               user.Role,
	})
}
