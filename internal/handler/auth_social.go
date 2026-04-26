package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/service"
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
		ServerError(c, M(c, i18n.KeyOAuthSocialNotConfigured))
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
		ServerError(c, M(c, i18n.KeyOAuthSocialNotConfigured))
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
		BadRequest(c, M(c, i18n.KeyOAuthTicketRequired))
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
		ServerError(c, M(c, i18n.KeyOAuthExchangeFailed))
		return
	}

	writeAuthCookies(c, tokenPair)
	Success(c, gin.H{
		"access_token":       tokenPair.AccessToken,
		"refresh_token":      tokenPair.RefreshToken,
		"expires_at":         tokenPair.ExpiresAt,
		"refresh_expires_at": tokenPair.RefreshExpiresAt,
		"return_to":          returnTo,
	})
}

// OnboardOAuth completes social registration after the user fills in missing profile data.
func (h *AuthHandler) OnboardOAuth(c *gin.Context) {
	var req onboardOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyOAuthInvalidOnboarding, err.Error()))
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
			ServerError(c, M(c, i18n.KeyOAuthOnboardingFailed))
		}
		return
	}

	writeAuthCookies(c, tokenPair)
	Success(c, gin.H{
		"access_token":       tokenPair.AccessToken,
		"refresh_token":      tokenPair.RefreshToken,
		"expires_at":         tokenPair.ExpiresAt,
		"refresh_expires_at": tokenPair.RefreshExpiresAt,
		"return_to":          returnTo,
	})
}

// ListIdentities returns the current user's third-party binding status.
func (h *AuthHandler) ListIdentities(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		Unauthorized(c, M(c, i18n.KeyAuthUserNotFound))
		return
	}
	providers, err := h.socialAuthSvc.ListUserIdentities(c.Request.Context(), userID)
	if err != nil {
		ServerError(c, M(c, i18n.KeyOAuthListIdentitiesFailed))
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
			NotFound(c, M(c, i18n.KeyOAuthIdentityNotFound))
		default:
			ServerError(c, M(c, i18n.KeyOAuthUnlinkIdentityFailed))
		}
		return
	}
	Success(c, nil)
}

// SetPassword lets a social-login-only account create its first local password.
func (h *AuthHandler) SetPassword(c *gin.Context) {
	var req setPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyAuthInvalidPassword, err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	if err := h.authSvc.SetPassword(c.Request.Context(), userID, req.NewPassword); err != nil {
		ServerError(c, M(c, i18n.KeyAuthSetPasswordFailed))
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

// accessTokenCookieName and refreshTokenCookieName are the cookies the web UI
// relies on exclusively — tokens are never exposed to JavaScript. Mobile and
// API clients can still read the JSON body of /auth/login + /auth/refresh.
const (
	accessTokenCookieName  = "access_token"
	refreshTokenCookieName = "refresh_token"
	// refreshTokenCookiePath scopes the refresh cookie to auth endpoints so
	// non-auth requests don't transmit a long-lived credential. Logout lives
	// under the same prefix so the clear-cookie write round-trips correctly.
	refreshTokenCookiePath = "/api/v1/auth"
)

func writeAccessTokenCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := maxAgeFromExpiry(expiresAt)
	if token == "" {
		maxAge = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     accessTokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

// writeRefreshTokenCookie sets (or clears, if token is empty) the HttpOnly
// refresh cookie. SameSite=Strict prevents the refresh token from being sent
// on cross-site navigations, which combined with the HttpOnly flag and the
// narrow /api/v1/auth path keeps the long-lived credential out of reach of
// both XSS and CSRF.
func writeRefreshTokenCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := maxAgeFromExpiry(expiresAt)
	if token == "" {
		maxAge = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     refreshTokenCookiePath,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteStrictMode,
	})
}

// writeAuthCookies rotates both the access and refresh cookies after a
// successful login, refresh, OAuth exchange, or credential reset. Callers
// pass the refresh-token expiry separately because in the current config it
// lives much longer than the access token.
func writeAuthCookies(c *gin.Context, tokenPair *service.TokenPair) {
	if tokenPair == nil {
		return
	}
	writeAccessTokenCookie(c, tokenPair.AccessToken, tokenPair.ExpiresAt)
	refreshExpiry := tokenPair.RefreshExpiresAt
	if refreshExpiry.IsZero() {
		refreshExpiry = tokenPair.ExpiresAt
	}
	writeRefreshTokenCookie(c, tokenPair.RefreshToken, refreshExpiry)
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
