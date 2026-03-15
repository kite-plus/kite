package api

import (
	"errors"
	"net/http"

	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

const adminSessionCookieName = "kite_admin_session"

type AdminAuthHandler struct {
	authService *service.AdminAuthService
}

func NewAdminAuthHandler(authService *service.AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{authService: authService}
}

func (h *AdminAuthHandler) Login(c *gin.Context) {
	var input service.AdminLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	rawToken, currentUser, err := h.authService.Login(input, service.AdminSessionMeta{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		handleAdminAuthError(c, err)
		return
	}

	maxAge := int(h.authService.SessionTTL().Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(adminSessionCookieName, rawToken, maxAge, "/", "", c.Request.TLS != nil, true)
	Success(c, currentUser)
}

func (h *AdminAuthHandler) Me(c *gin.Context) {
	rawToken, _ := c.Cookie(adminSessionCookieName)
	currentUser, err := h.authService.GetCurrent(rawToken)
	if err != nil {
		handleAdminAuthError(c, err)
		return
	}

	Success(c, currentUser)
}

func (h *AdminAuthHandler) Logout(c *gin.Context) {
	rawToken, _ := c.Cookie(adminSessionCookieName)
	if err := h.authService.Logout(rawToken); err != nil {
		handleAdminAuthError(c, err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(adminSessionCookieName, "", -1, "/", "", c.Request.TLS != nil, true)
	Success(c, gin.H{"logged_out": true})
}

func handleAdminAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAdminAuthDisabled):
		Error(c, http.StatusServiceUnavailable, http.StatusServiceUnavailable, "admin auth is disabled")
	case errors.Is(err, service.ErrInvalidAdminCredentials):
		Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "invalid username or password")
	case errors.Is(err, service.ErrAdminUnauthorized):
		Error(c, http.StatusUnauthorized, http.StatusUnauthorized, "unauthorized")
	default:
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
