package handler

import (
	"strings"

	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

// SettingsHandler handles system settings HTTP requests.
type SettingsHandler struct {
	settingRepo *repo.SettingRepo
	defaults    map[string]string
}

func NewSettingsHandler(settingRepo *repo.SettingRepo, defaults map[string]string) *SettingsHandler {
	if defaults == nil {
		defaults = map[string]string{}
	}
	return &SettingsHandler{settingRepo: settingRepo, defaults: defaults}
}

// Get returns all system settings.
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.settingRepo.GetAll(c.Request.Context())
	if err != nil {
		ServerError(c, "failed to get settings")
		return
	}
	merged := service.ResolveSettings(h.defaults, settings)
	Success(c, merged)
}

type updateSettingsRequest struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

// Update bulk-updates system settings.
func (h *SettingsHandler) Update(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid settings data: "+err.Error())
		return
	}

	if raw, ok := req.Settings[service.UploadPathPatternSettingKey]; ok {
		pattern, err := service.NormalizeUploadPathPattern(raw)
		if err != nil {
			BadRequest(c, "invalid upload.path_pattern: "+err.Error())
			return
		}
		req.Settings[service.UploadPathPatternSettingKey] = strings.TrimSpace(pattern)
	}
	if raw, ok := req.Settings[service.UploadMaxFileSizeMBSettingKey]; ok {
		normalized, err := service.NormalizeUploadMaxFileSizeMB(raw)
		if err != nil {
			BadRequest(c, "invalid upload.max_file_size_mb: "+err.Error())
			return
		}
		req.Settings[service.UploadMaxFileSizeMBSettingKey] = normalized
	}

	for _, key := range []string{
		service.SiteNameSettingKey,
		service.SiteURLSettingKey,
		service.SiteTitleSettingKey,
		service.SiteKeywordsSettingKey,
		service.SiteDescriptionSettingKey,
		service.SiteFaviconURLSettingKey,
		service.SiteHeaderBrandSettingKey,
		service.SiteHeaderNavGitHubURLSettingKey,
		service.SiteFooterTextSettingKey,
		service.SiteFooterCopyrightSettingKey,
	} {
		raw, ok := req.Settings[key]
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" && (key == service.SiteNameSettingKey || key == service.SiteTitleSettingKey || key == service.SiteHeaderBrandSettingKey) {
			BadRequest(c, "invalid "+key+": cannot be empty")
			return
		}
		req.Settings[key] = trimmed
	}

	if err := h.settingRepo.SetBatch(c.Request.Context(), req.Settings); err != nil {
		ServerError(c, "failed to update settings")
		return
	}

	Success(c, nil)
}
