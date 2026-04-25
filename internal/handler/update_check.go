package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/service"
)

// UpdateCheckHandler exposes the upstream-version probe to the admin UI. The
// underlying service caches its answer for several hours so this handler is
// safe to call on every dashboard mount.
type UpdateCheckHandler struct {
	svc *service.UpdateCheckService
}

func NewUpdateCheckHandler(svc *service.UpdateCheckService) *UpdateCheckHandler {
	return &UpdateCheckHandler{svc: svc}
}

// Check returns the cached UpdateInfo. The `?force=1` query forces a fresh
// fetch — useful when the admin clicks a "check now" button after just
// publishing a release. Upstream errors degrade to a 200 with empty Latest
// so the dashboard doesn't have to special-case "GitHub is unreachable".
func (h *UpdateCheckHandler) Check(c *gin.Context) {
	force := c.Query("force") == "1" || c.Query("force") == "true"
	info, err := h.svc.Check(c.Request.Context(), force)
	if err != nil {
		// We deliberately don't surface a 500 here. The admin already
		// has bigger problems if GitHub is unreachable from the server,
		// and a noisy red banner on every dashboard load helps no one.
		Success(c, gin.H{
			"current":    "",
			"latest":     "",
			"has_update": false,
			"error":      err.Error(),
		})
		return
	}
	Success(c, info)
}
