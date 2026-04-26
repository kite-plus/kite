package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/service"
)

// SetupTOTP starts (or restarts) 2FA enrollment for the authenticated
// user. Returns the otpauth:// URI so the client can render a QR code,
// plus the raw Base32 secret as a fallback for users who cannot scan.
// Calling this while 2FA is already enabled is a 409 — the client
// must disable first to prevent accidentally overwriting a working
// setup.
func (h *AuthHandler) SetupTOTP(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	enrollment, err := h.authSvc.BeginTOTPEnrollment(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrTOTPAlreadyEnabled) {
			Fail(c, http.StatusConflict, 40900, err.Error())
			return
		}
		ServerError(c, M(c, i18n.KeyTwoFAStartSetupFailed))
		return
	}
	Success(c, gin.H{
		"secret": enrollment.Secret,
		"uri":    enrollment.URI,
	})
}

type enableTOTPRequest struct {
	Code string `json:"code" binding:"required"`
}

// EnableTOTP confirms the pending secret by validating the user's
// first code. On success totp_enabled flips to true and every
// outstanding session on the account is revoked via token_version.
func (h *AuthHandler) EnableTOTP(c *gin.Context) {
	var req enableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyTwoFACodeRequired))
		return
	}
	userID := c.GetString(middleware.ContextKeyUserID)
	if err := h.authSvc.ConfirmTOTPEnrollment(c.Request.Context(), userID, req.Code); err != nil {
		switch {
		case errors.Is(err, service.ErrTOTPAlreadyEnabled):
			Fail(c, http.StatusConflict, 40900, err.Error())
		case errors.Is(err, service.ErrTOTPNotEnrolled):
			Fail(c, http.StatusBadRequest, 40000, err.Error())
		case errors.Is(err, service.ErrTOTPInvalidCode):
			Fail(c, http.StatusBadRequest, 40001, err.Error())
		default:
			ServerError(c, M(c, i18n.KeyTwoFAEnableFailed))
		}
		return
	}
	// Enabling rotates token_version, which instantly invalidates the
	// cookie the user is holding. Re-issue so the immediate next
	// request doesn't bounce them to /login.
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		ServerError(c, M(c, i18n.KeyAuthReloadUserFailed))
		return
	}
	pair, err := h.authSvc.IssueTokenPair(user)
	if err != nil {
		ServerError(c, M(c, i18n.KeyAuthRefreshSessionFailed))
		return
	}
	writeAuthCookies(c, pair)
	Success(c, gin.H{"totp_enabled": true})
}

type disableTOTPRequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// DisableTOTP turns 2FA off. Both the password and a current TOTP
// code are required — password alone would let a stolen session strip
// the second factor, code alone would let anyone with temporary
// phone access do the same.
func (h *AuthHandler) DisableTOTP(c *gin.Context) {
	var req disableTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyTwoFAPasswordAndCodeRequired))
		return
	}
	userID := c.GetString(middleware.ContextKeyUserID)
	if err := h.authSvc.DisableTOTP(c.Request.Context(), userID, req.Password, req.Code); err != nil {
		switch {
		case errors.Is(err, service.ErrTOTPNotEnabled):
			Fail(c, http.StatusBadRequest, 40000, err.Error())
		case errors.Is(err, service.ErrTOTPInvalidCode):
			Fail(c, http.StatusBadRequest, 40001, err.Error())
		case errors.Is(err, service.ErrPasswordMismatch):
			Fail(c, http.StatusBadRequest, 40002, err.Error())
		case errors.Is(err, service.ErrLocalPasswordNotSet):
			Fail(c, http.StatusBadRequest, 40003, err.Error())
		default:
			ServerError(c, M(c, i18n.KeyTwoFADisableFailed))
		}
		return
	}
	// Same reasoning as EnableTOTP: the session cookie the caller
	// is holding was just invalidated. Re-issue to stay logged in.
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		ServerError(c, M(c, i18n.KeyAuthReloadUserFailed))
		return
	}
	pair, err := h.authSvc.IssueTokenPair(user)
	if err != nil {
		ServerError(c, M(c, i18n.KeyAuthRefreshSessionFailed))
		return
	}
	writeAuthCookies(c, pair)
	Success(c, gin.H{"totp_enabled": false})
}

type verifyTOTPRequest struct {
	ChallengeToken string `json:"challenge_token" binding:"required"`
	Code           string `json:"code" binding:"required"`
}

// VerifyTOTP exchanges a 2FA challenge token + fresh code for a full
// token pair. This is the endpoint the login page hits after the
// password step returned pending_2fa=true.
func (h *AuthHandler) VerifyTOTP(c *gin.Context) {
	var req verifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyTwoFAChallengeRequired))
		return
	}
	pair, err := h.authSvc.ConsumeTOTPChallenge(c.Request.Context(), req.ChallengeToken, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrChallengeInvalid):
			Unauthorized(c, err.Error())
		case errors.Is(err, service.ErrUserInactive):
			Unauthorized(c, err.Error())
		case errors.Is(err, service.ErrTOTPNotEnabled):
			Fail(c, http.StatusBadRequest, 40000, err.Error())
		case errors.Is(err, service.ErrTOTPInvalidCode):
			Fail(c, http.StatusBadRequest, 40001, err.Error())
		default:
			ServerError(c, M(c, i18n.KeyTwoFAVerifyFailed))
		}
		return
	}
	writeAuthCookies(c, pair)
	Success(c, pair)
}
