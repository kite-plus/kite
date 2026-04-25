package router

import (
	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/handler"
)

// registerUpdateCheckAdmin wires the admin-only endpoint that reports whether
// a newer release is available upstream. The parent group must already carry
// middleware.Auth and middleware.AdminOnly.
func registerUpdateCheckAdmin(admin *gin.RouterGroup, h *handler.UpdateCheckHandler) {
	admin.GET("/admin/system/update-check", h.Check)
}
