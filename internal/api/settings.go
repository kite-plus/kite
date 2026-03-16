package api

import (
	"net/http"

	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// SettingsHandler 设置 API 处理器
type SettingsHandler struct {
	settingsService *service.SettingsService
}

func NewSettingsHandler(settingsService *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService}
}

// Get 获取全部设置
func (h *SettingsHandler) Get(c *gin.Context) {
	settings := h.settingsService.Get()
	Success(c, settings)
}

// Update 更新全部设置
func (h *SettingsHandler) Update(c *gin.Context) {
	var input service.AllSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	result, err := h.settingsService.Update(input)
	if err != nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, result)
}
