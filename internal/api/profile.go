package api

import (
	"net/http"

	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// ProfileHandler 个人资料 API 处理器
type ProfileHandler struct {
	settingsService *service.SettingsService
	authService     *service.AdminAuthService
	settingsRepo    *repo.SettingsRepository
}

func NewProfileHandler(settingsService *service.SettingsService, authService *service.AdminAuthService, settingsRepo *repo.SettingsRepository) *ProfileHandler {
	return &ProfileHandler{
		settingsService: settingsService,
		authService:     authService,
		settingsRepo:    settingsRepo,
	}
}

// Get 获取当前管理员个人资料
func (h *ProfileHandler) Get(c *gin.Context) {
	profile := h.settingsService.GetProfile()
	Success(c, profile)
}

// Update 更新管理员个人资料
func (h *ProfileHandler) Update(c *gin.Context) {
	var input service.ProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	profile, err := h.settingsService.UpdateProfile(input)
	if err != nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}
	Success(c, profile)
}

// ChangePassword 修改密码
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	var input service.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.authService.ChangePassword(input, h.settingsRepo); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	Success(c, gin.H{"changed": true})
}
