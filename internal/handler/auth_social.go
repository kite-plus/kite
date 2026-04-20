package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/middleware"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const oauthStateCookieName = "kite_oauth_state"

type exchangeOAuthRequest struct {
	Ticket string `json:"ticket" binding:"required"`
}

type onboardOAuthRequest struct {
	Ticket   string `json:"ticket" binding:"required"`
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
}

type setPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

// StartOAuth redirects the browser to the selected provider.
func (h *AuthHandler) StartOAuth(c *gin.Context) {
	if h.socialAuthSvc == nil {
		ServerError(c, "social login is not configured")
		return
	}

	provider := c.Param("provider")
	mode := c.DefaultQuery("mode", service.OAuthModeLogin)
	currentUserID := ""
	if mode == service.OAuthModeBind {
		currentUserID = h.extractCurrentUserID(c)
	}

	result, err := h.socialAuthSvc.PrepareAuthRedirect(
		c.Request.Context(),
		provider,
		mode,
		c.Query("return_to"),
		currentUserID,
	)
	if err != nil {
		redirectSocialError(c, mode, c.Query("return_to"), err.Error())
		return
	}

	writeOAuthStateCookie(c, result.CookieValue, result.ExpiresAt)
	c.Redirect(http.StatusFound, result.RedirectURL)
}

// OAuthCallback handles the provider callback and redirects back into the SPA.
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
	if h.socialAuthSvc == nil {
		ServerError(c, "social login is not configured")
		return
	}

	provider := c.Param("provider")
	cookieValue, _ := c.Cookie(oauthStateCookieName)
	mode, returnTo, _ := h.socialAuthSvc.InspectStateCookie(cookieValue)
	if mode == "" {
		mode = service.OAuthModeLogin
	}
	if returnTo == "" {
		if mode == service.OAuthModeBind {
			returnTo = "/user/profile"
		} else {
			returnTo = "/login"
		}
	}
	defer clearOAuthStateCookie(c)

	if providerError := strings.TrimSpace(c.Query("error")); providerError != "" {
		msg := strings.TrimSpace(c.Query("error_description"))
		if msg == "" {
			msg = providerError
		}
		redirectSocialError(c, mode, returnTo, msg)
		return
	}

	currentUserID := ""
	if mode == service.OAuthModeBind {
		currentUserID = h.extractCurrentUserID(c)
	}
	redirectTo, err := h.socialAuthSvc.HandleCallback(
		c.Request.Context(),
		provider,
		c.Query("code"),
		c.Query("state"),
		cookieValue,
		currentUserID,
	)
	if err != nil {
		redirectSocialError(c, mode, returnTo, err.Error())
		return
	}

	c.Redirect(http.StatusFound, redirectTo)
}

// ExchangeOAuth converts a short-lived social login ticket into normal API tokens.
func (h *AuthHandler) ExchangeOAuth(c *gin.Context) {
	var req exchangeOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "ticket is required")
		return
	}

	tokenPair, _, returnTo, err := h.socialAuthSvc.ExchangeLoginTicket(c.Request.Context(), req.Ticket)
	if err != nil {
		if errors.Is(err, service.ErrOAuthTicketInvalid) {
			BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrUserInactive) {
			Forbidden(c, err.Error())
			return
		}
		ServerError(c, "exchange oauth ticket failed")
		return
	}

	writeAccessTokenCookie(c, tokenPair.AccessToken, tokenPair.ExpiresAt)
	Success(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"return_to":     returnTo,
	})
}

// OnboardOAuth completes social registration after the user fills in missing profile data.
func (h *AuthHandler) OnboardOAuth(c *gin.Context) {
	var req onboardOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid onboarding data: "+err.Error())
		return
	}

	tokenPair, _, returnTo, err := h.socialAuthSvc.CompleteOnboarding(
		c.Request.Context(),
		req.Ticket,
		req.Username,
		req.Email,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOAuthTicketInvalid):
			BadRequest(c, err.Error())
		case errors.Is(err, service.ErrRegistrationClosed):
			Forbidden(c, err.Error())
		case errors.Is(err, service.ErrUserExists):
			Fail(c, http.StatusConflict, 40900, err.Error())
		default:
			ServerError(c, "oauth onboarding failed")
		}
		return
	}

	writeAccessTokenCookie(c, tokenPair.AccessToken, tokenPair.ExpiresAt)
	Success(c, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"return_to":     returnTo,
	})
}

// ListIdentities returns the current user's third-party binding status.
func (h *AuthHandler) ListIdentities(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		Unauthorized(c, "user not found")
		return
	}
	providers, err := h.socialAuthSvc.ListUserIdentities(c.Request.Context(), userID)
	if err != nil {
		ServerError(c, "failed to list identities")
		return
	}

	Success(c, gin.H{
		"has_local_password": user.HasLocalPassword,
		"providers":          providers,
	})
}

// UnlinkIdentity removes the current user's binding for one provider.
func (h *AuthHandler) UnlinkIdentity(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	provider := c.Param("provider")
	if err := h.socialAuthSvc.UnlinkIdentity(c.Request.Context(), userID, provider); err != nil {
		switch {
		case errors.Is(err, service.ErrOAuthLastLoginMethod):
			Fail(c, http.StatusBadRequest, 40020, err.Error())
		case errors.Is(err, service.ErrOAuthProviderUnsupported):
			BadRequest(c, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			NotFound(c, "identity not found")
		default:
			ServerError(c, "failed to unlink identity")
		}
		return
	}
	Success(c, nil)
}

// SetPassword lets a social-login-only account create its first local password.
func (h *AuthHandler) SetPassword(c *gin.Context) {
	var req setPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid password data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	if err := h.authSvc.SetPassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		ServerError(c, "set password failed")
		return
	}
	Success(c, nil)
}

func (h *AuthHandler) extractCurrentUserID(c *gin.Context) string {
	token := extractRequestToken(c)
	if token == "" {
		return ""
	}
	claims, err := h.authSvc.ValidateToken(token)
	if err != nil {
		return ""
	}
	return claims.UserID
}

func extractRequestToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if cookie, err := c.Cookie("access_token"); err == nil {
		return strings.TrimSpace(cookie)
	}
	return ""
}

func writeOAuthStateCookie(c *gin.Context, value string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAgeFromExpiry(expiresAt),
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthStateCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func writeAccessTokenCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := maxAgeFromExpiry(expiresAt)
	if token == "" {
		maxAge = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func isSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")), "https")
}

func maxAgeFromExpiry(expiresAt time.Time) int {
	seconds := int(time.Until(expiresAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func redirectSocialError(c *gin.Context, mode, returnTo, message string) {
	target := "/login"
	values := map[string]string{
		"oauth_error": strings.TrimSpace(message),
	}
	if mode == service.OAuthModeBind {
		target = returnTo
		values["social_status"] = "error"
	}
	c.Redirect(http.StatusFound, appendRedirectQuery(target, values))
}

func appendRedirectQuery(target string, values map[string]string) string {
	fallback := "/login"
	if strings.TrimSpace(target) == "" {
		target = fallback
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return fallback
	}
	query := parsed.Query()
	for key, value := range values {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
