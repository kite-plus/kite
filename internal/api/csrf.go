package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32
)

// generateCSRFToken 生成随机 CSRF token
func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CSRFMiddleware 基于 double-submit cookie 模式的 CSRF 防护
// 写请求（POST/PUT/PATCH/DELETE）必须携带 X-CSRF-Token header，值与 csrf_token cookie 一致
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 确保存在 CSRF cookie；若不存在则生成
		token, err := c.Cookie(csrfCookieName)
		if err != nil || token == "" {
			token, err = generateCSRFToken()
			if err != nil {
				Error(c, http.StatusInternalServerError, 500, "failed to generate csrf token")
				c.Abort()
				return
			}
			c.SetCookie(csrfCookieName, token, 86400*7, "/", "", false, false) // non-HttpOnly，JS 可读
		}

		// 读请求不验证
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// 写请求：验证 header 与 cookie 匹配
		headerToken := c.GetHeader(csrfHeaderName)
		if headerToken == "" || headerToken != token {
			Error(c, http.StatusForbidden, 403, "csrf token invalid")
			c.Abort()
			return
		}

		c.Next()
	}
}
