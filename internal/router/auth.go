package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/handler"
	"github.com/kite-plus/kite/internal/repo"
)

// registerAuthPublic wires unauthenticated authentication endpoints under
// /api/v1/auth with a per-IP rate limit to slow credential stuffing.
//
// /auth/login and /auth/refresh moved to internal/api (typed/OpenAPI). The
// gin entries are intentionally absent here — re-adding them would clash
// with the huma registrations made by api.Register.
func registerAuthPublic(v1 *gin.RouterGroup, h *handler.AuthHandler, settingRepo *repo.SettingRepo) {
	g := v1.Group("/auth")
	g.Use(authRateLimit(settingRepo))

	g.GET("/options", h.Options)
	g.GET("/oauth/:provider/start", h.StartOAuth)
	g.GET("/oauth/:provider/callback", h.OAuthCallback)
	g.POST("/register", h.Register)
	g.POST("/oauth/exchange", h.ExchangeOAuth)
	g.POST("/oauth/onboard", h.OnboardOAuth)
	// /2fa/verify is public because the caller only holds a challenge
	// token at this point — they've passed the password step but don't
	// yet have a session cookie to present to middleware.Auth.
	g.POST("/2fa/verify", h.VerifyTOTP)
	// Forgot-password flow — both endpoints are public by design
	// (anonymous users call them) and inherit the shared auth rate
	// limit above, so an attacker can't mass-spam the mailer.
	g.POST("/password-reset/request", h.RequestPasswordReset)
	g.POST("/password-reset/confirm", h.ConfirmPasswordReset)
}

// registerAuthAuthed wires authenticated profile and credential endpoints.
// The parent group is expected to already carry middleware.Auth.
//
// GET /profile and POST /auth/logout moved to internal/api (typed/OpenAPI).
// The gin entries are intentionally absent here — duplicates would crash
// gin at boot.
func registerAuthAuthed(authed *gin.RouterGroup, h *handler.AuthHandler) {
	authed.PUT("/profile", h.UpdateProfile)
	authed.GET("/auth/identities", h.ListIdentities)
	authed.POST("/auth/change-password", h.ChangePassword)
	authed.POST("/auth/set-password", h.SetPassword)
	authed.POST("/auth/first-login-reset", h.FirstLoginReset)
	authed.POST("/auth/email-change/request", h.RequestEmailChange)
	authed.POST("/auth/email-change/confirm", h.ConfirmEmailChange)
	authed.DELETE("/auth/identities/:provider", h.UnlinkIdentity)
	authed.POST("/auth/2fa/setup", h.SetupTOTP)
	authed.POST("/auth/2fa/enable", h.EnableTOTP)
	authed.POST("/auth/2fa/disable", h.DisableTOTP)
}

// registerAuthAdmin wires admin-only provider configuration endpoints.
func registerAuthAdmin(admin *gin.RouterGroup, h *handler.OAuthProviderAdminHandler) {
	admin.GET("/admin/auth/providers", h.List)
	admin.PUT("/admin/auth/providers/:provider", h.Update)
}
