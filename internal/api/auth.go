package api

import (
	"errors"
	"net/http"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证相关的 HTTP 处理器。
type AuthHandler struct {
	authSvc  *service.AuthService
	userRepo *repo.UserRepo
}

func NewAuthHandler(authSvc *service.AuthService, userRepo *repo.UserRepo) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userRepo: userRepo}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "username and password are required")
		return
	}

	tokenPair, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserInactive) {
			unauthorized(c, err.Error())
			return
		}
		serverError(c, "login failed")
		return
	}

	// 同时设置 cookie 供 Web 界面使用
	c.SetCookie("access_token", tokenPair.AccessToken, 7200, "/", "", false, true)

	success(c, tokenPair)
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

// Register 用户注册。
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid registration data: "+err.Error())
		return
	}

	user, err := h.authSvc.Register(c.Request.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrRegistrationClosed) {
			forbidden(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserExists) {
			fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
		serverError(c, "registration failed")
		return
	}

	created(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken 刷新 access token。
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "refresh_token is required")
		return
	}

	tokenPair, err := h.authSvc.RefreshToken(req.RefreshToken)
	if err != nil {
		unauthorized(c, "invalid refresh token")
		return
	}

	c.SetCookie("access_token", tokenPair.AccessToken, 7200, "/", "", false, true)
	success(c, tokenPair)
}

// Logout 登出，清除 cookie。
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	success(c, nil)
}

// GetProfile 获取当前登录用户信息。
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		unauthorized(c, "user not found")
		return
	}

	success(c, gin.H{
		"user_id":              user.ID,
		"username":             user.Username,
		"nickname":             user.Nickname,
		"email":                user.Email,
		"avatar_url":           user.AvatarURL,
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

// UpdateProfile 当前登录用户更新自己的用户名、昵称、邮箱与头像。
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid profile data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.authSvc.UpdateProfile(c.Request.Context(), userID, req.Username, req.Nickname, req.Email, req.AvatarURL)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
		serverError(c, "update profile failed")
		return
	}

	success(c, gin.H{
		"user_id":    user.ID,
		"username":   user.Username,
		"nickname":   user.Nickname,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
		"role":       user.Role,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=64"`
}

// ChangePassword 当前登录用户修改自己的密码。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid password data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrPasswordMismatch) {
			fail(c, http.StatusBadRequest, 40010, err.Error())
			return
		}
		serverError(c, "change password failed")
		return
	}

	success(c, nil)
}

type firstLoginResetRequest struct {
	NewUsername string `json:"new_username" binding:"required,min=3,max=32"`
	NewEmail    string `json:"new_email" binding:"required,email"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

// FirstLoginReset 首次登录强制重置账号与密码。
// 仅当用户 PasswordMustChange=true 时允许；成功后返回新的 token pair。
func (h *AuthHandler) FirstLoginReset(c *gin.Context) {
	var req firstLoginResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid reset data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	tokenPair, err := h.authSvc.ResetFirstLoginCredentials(
		c.Request.Context(), userID, req.NewUsername, req.NewEmail, req.NewPassword,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
		badRequest(c, err.Error())
		return
	}

	c.SetCookie("access_token", tokenPair.AccessToken, 7200, "/", "", false, true)
	success(c, tokenPair)
}
