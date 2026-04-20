package router

import (
	"github.com/amigoer/kite/internal/handler"
	"github.com/gin-gonic/gin"
)

// registerAuthPublic wires unauthenticated authentication endpoints under
// /api/v1/auth with a per-IP rate limit to slow credential stuffing.
func registerAuthPublic(v1 *gin.RouterGroup, h *handler.AuthHandler) {
	g := v1.Group("/auth")
	g.Use(authRateLimit(20))

	g.GET("/options", h.Options)
	g.GET("/oauth/:provider/start", h.StartOAuth)
	g.GET("/oauth/:provider/callback", h.OAuthCallback)
	g.POST("/login", h.Login)
	g.POST("/register", h.Register)
	g.POST("/refresh", h.RefreshToken)
	g.POST("/oauth/exchange", h.ExchangeOAuth)
	g.POST("/oauth/onboard", h.OnboardOAuth)
}

// registerAuthAuthed wires authenticated profile and credential endpoints.
// The parent group is expected to already carry middleware.Auth.
func registerAuthAuthed(authed *gin.RouterGroup, h *handler.AuthHandler) {
	authed.GET("/profile", h.GetProfile)
	authed.PUT("/profile", h.UpdateProfile)
	authed.GET("/auth/identities", h.ListIdentities)
	authed.POST("/auth/logout", h.Logout)
	authed.POST("/auth/change-password", h.ChangePassword)
	authed.POST("/auth/set-password", h.SetPassword)
	authed.POST("/auth/first-login-reset", h.FirstLoginReset)
	authed.DELETE("/auth/identities/:provider", h.UnlinkIdentity)
}

// registerAuthAdmin wires admin-only provider configuration endpoints.
func registerAuthAdmin(admin *gin.RouterGroup, h *handler.OAuthProviderAdminHandler) {
	admin.GET("/admin/auth/providers", h.List)
	admin.PUT("/admin/auth/providers/:provider", h.Update)
}
