package api

import (
	"errors"

	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminAuthMiddleware struct {
	authService *service.AdminAuthService
}

func NewAdminAuthMiddleware(authService *service.AdminAuthService) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{authService: authService}
}

func (m *AdminAuthMiddleware) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m == nil || m.authService == nil || !m.authService.IsEnabled() {
			c.Next()
			return
		}

		rawToken, err := c.Cookie(adminSessionCookieName)
		if err != nil {
			Error(c, 401, 401, "unauthorized")
			c.Abort()
			return
		}

		currentUser, err := m.authService.GetCurrent(rawToken)
		if err != nil {
			if errors.Is(err, service.ErrAdminUnauthorized) {
				Error(c, 401, 401, "unauthorized")
				c.Abort()
				return
			}
			Error(c, 500, 500, "internal server error")
			c.Abort()
			return
		}

		c.Set("admin_current_user", currentUser)
		c.Next()
	}
}
