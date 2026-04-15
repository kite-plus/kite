package middleware

import (
	"net/http"
	"strings"

	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyRole     = "role"
)

// Auth 认证中间件，支持 JWT 和 API Token 两种认证方式。
func Auth(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40100,
				"message": "authorization required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 优先尝试 JWT 认证
		claims, err := authSvc.ValidateToken(token)
		if err == nil {
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUsername, claims.Username)
			c.Set(ContextKeyRole, claims.Role)
			c.Next()
			return
		}

		// JWT 失败则尝试 API Token 认证
		userID, err := authSvc.ValidateAPIToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "invalid or expired token",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeyRole, "user") // API Token 默认 user 角色
		c.Next()
	}
}

// AdminOnly 管理员权限检查中间件，必须在 Auth 之后使用。
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(ContextKeyRole)
		if !exists || role.(string) != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40300,
				"message": "admin access required",
				"data":    nil,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// extractToken 从请求中提取 token。
// 仅支持 Authorization header 和 cookie。
// 不支持 query 参数，避免 token 泄露到浏览器历史、访问日志、Referer 头。
func extractToken(c *gin.Context) string {
	// Authorization: Bearer <token>
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Cookie
	if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
		return cookie
	}

	return ""
}
