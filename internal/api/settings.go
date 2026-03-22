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

// GetNavMenus 获取导航菜单
func (h *SettingsHandler) GetNavMenus(c *gin.Context) {
	menus := h.settingsService.GetNavMenus()
	if menus == nil {
		menus = []service.NavMenuItem{}
	}
	Success(c, menus)
}

// SaveNavMenus 保存导航菜单
func (h *SettingsHandler) SaveNavMenus(c *gin.Context) {
	var menus []service.NavMenuItem
	if err := c.ShouldBindJSON(&menus); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	if err := h.settingsService.SaveNavMenus(menus); err != nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, menus)
}
