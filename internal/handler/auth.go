package handler

import (
	"errors"
	"net/http"
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
	userRepo                 *repo.UserRepo
	settingRepo              *repo.SettingRepo
	allowRegistrationDefault bool
	uploadMaxFileSizeDefault int64
}

func NewAuthHandler(
	authSvc *service.AuthService,
	socialAuthSvc *service.SocialAuthService,
	oauthConfigSvc *service.OAuthConfigService,
	userRepo *repo.UserRepo,
	settingRepo *repo.SettingRepo,
	allowRegistrationDefault bool,
	uploadMaxFileSizeDefault int64,
) *AuthHandler {
	return &AuthHandler{
		authSvc:                  authSvc,
		socialAuthSvc:            socialAuthSvc,
		oauthConfigSvc:           oauthConfigSvc,
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

	// Also set a cookie for the web UI.
	writeAccessTokenCookie(c, tokenPair.AccessToken, tokenPair.ExpiresAt)

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
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken exchanges a refresh token for a new access token.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "refresh_token is required")
		return
	}

	tokenPair, err := h.authSvc.RefreshToken(req.RefreshToken)
	if err != nil {
		Unauthorized(c, "invalid refresh token")
		return
	}

	writeAccessTokenCookie(c, tokenPair.AccessToken, tokenPair.ExpiresAt)
	Success(c, tokenPair)
}

// Logout clears the access token cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	writeAccessTokenCookie(c, "", time.Unix(0, 0))
	Success(c, nil)
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
	Username  string  `json:"username" binding:"required,min=3,max=32"`
	Nickname  *string `json:"nickname" binding:"omitempty,max=32"`
	Email     string  `json:"email" binding:"required,email"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,max=512"`
}

// UpdateProfile lets the current user update their username, nickname, email, and avatar.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid profile data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.authSvc.UpdateProfile(c.Request.Context(), userID, req.Username, req.Nickname, req.Email, req.AvatarURL)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			Fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
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

	writeAccessTokenCookie(c, tokenPair.AccessToken, tokenPair.ExpiresAt)
	Success(c, tokenPair)
}
