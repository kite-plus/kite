package api

import (
	"net/http"

	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	systemService *service.SystemService
}

func NewHealthHandler(systemService *service.SystemService) *HealthHandler {
	return &HealthHandler{systemService: systemService}
}

func (h *HealthHandler) Get(c *gin.Context) {
	if h == nil || h.systemService == nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "system service is unavailable")
		return
	}

	Success(c, h.systemService.HealthStatus())
}
