package handler

import (
	"errors"
	"strings"

	"github.com/amigoer/kite/internal/middleware"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SettingsHandler handles system settings HTTP requests.
type SettingsHandler struct {
	settingRepo *repo.SettingRepo
	userRepo    *repo.UserRepo
	emailSvc    *service.EmailService
	defaults    map[string]string
}

func NewSettingsHandler(settingRepo *repo.SettingRepo, userRepo *repo.UserRepo, emailSvc *service.EmailService, defaults map[string]string) *SettingsHandler {
	if defaults == nil {
		defaults = map[string]string{}
	}
	return &SettingsHandler{settingRepo: settingRepo, userRepo: userRepo, emailSvc: emailSvc, defaults: defaults}
}

// Get returns all system settings.
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.settingRepo.GetAll(c.Request.Context())
	if err != nil {
		ServerError(c, "failed to get settings")
		return
	}
	merged := service.ResolveSettings(h.defaults, settings)
	merged[service.SMTPPasswordConfiguredSettingKey] = "false"
	if strings.TrimSpace(settings[service.SMTPPasswordSettingKey]) != "" {
		merged[service.SMTPPasswordConfiguredSettingKey] = "true"
	}
	delete(merged, service.SMTPPasswordSettingKey)
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
	if raw, ok := req.Settings[service.DefaultQuotaSettingKey]; ok {
		normalized, err := service.NormalizeDefaultQuota(raw)
		if err != nil {
			BadRequest(c, "invalid "+service.DefaultQuotaSettingKey+": "+err.Error())
			return
		}
		req.Settings[service.DefaultQuotaSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.UploadMaxFileSizeMBSettingKey]; ok {
		normalized, err := service.NormalizeUploadMaxFileSizeMB(raw)
		if err != nil {
			BadRequest(c, "invalid upload.max_file_size_mb: "+err.Error())
			return
		}
		req.Settings[service.UploadMaxFileSizeMBSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.UploadDangerousExtensionRulesSettingKey]; ok {
		normalized, err := service.NormalizeDangerousExtensionRules(raw)
		if err != nil {
			BadRequest(c, "invalid upload.dangerous_extension_rules: "+err.Error())
			return
		}
		req.Settings[service.UploadDangerousExtensionRulesSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.UploadDangerousRenameSuffixSettingKey]; ok {
		normalized, err := service.NormalizeDangerousRenameSuffix(raw)
		if err != nil {
			BadRequest(c, "invalid upload.dangerous_rename_suffix: "+err.Error())
			return
		}
		req.Settings[service.UploadDangerousRenameSuffixSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPPortSettingKey]; ok {
		normalized, err := service.NormalizeSMTPPort(raw)
		if err != nil {
			BadRequest(c, "invalid "+service.SMTPPortSettingKey+": "+err.Error())
			return
		}
		req.Settings[service.SMTPPortSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPTLSSettingKey]; ok {
		normalized, err := service.NormalizeSMTPBool(raw)
		if err != nil {
			BadRequest(c, "invalid "+service.SMTPTLSSettingKey+": "+err.Error())
			return
		}
		req.Settings[service.SMTPTLSSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPFromSettingKey]; ok {
		normalized, err := service.NormalizeSMTPFrom(raw)
		if err != nil {
			BadRequest(c, "invalid "+service.SMTPFromSettingKey+": "+err.Error())
			return
		}
		req.Settings[service.SMTPFromSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPHostSettingKey]; ok {
		req.Settings[service.SMTPHostSettingKey] = service.NormalizeSMTPHost(raw)
	}
	if raw, ok := req.Settings[service.SMTPUsernameSettingKey]; ok {
		req.Settings[service.SMTPUsernameSettingKey] = service.NormalizeSMTPUsername(raw)
	}
	for _, key := range []string{
		service.AuthRateLimitPerMinuteSettingKey,
		service.GuestUploadRateLimitPerMinuteSettingKey,
	} {
		raw, ok := req.Settings[key]
		if !ok {
			continue
		}
		normalized, err := service.NormalizeRequestsPerMinute(raw)
		if err != nil {
			BadRequest(c, "invalid "+key+": "+err.Error())
			return
		}
		req.Settings[key] = normalized
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
	delete(req.Settings, service.SMTPPasswordConfiguredSettingKey)

	if err := h.settingRepo.SetBatch(c.Request.Context(), req.Settings); err != nil {
		ServerError(c, "failed to update settings")
		return
	}

	Success(c, nil)
}

func (h *SettingsHandler) TestEmail(c *gin.Context) {
	if h.emailSvc == nil || h.userRepo == nil {
		ServerError(c, "email service is not configured")
		return
	}

	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "invalid settings data: "+err.Error())
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Unauthorized(c, "user not found")
			return
		}
		ServerError(c, "failed to load current user")
		return
	}

	storedSettings, err := h.settingRepo.GetAll(c.Request.Context())
	if err != nil {
		ServerError(c, "failed to get settings")
		return
	}

	resolved := make(map[string]string, len(storedSettings)+len(req.Settings))
	for key, value := range storedSettings {
		resolved[key] = value
	}
	for key, value := range req.Settings {
		if key == service.SMTPPasswordConfiguredSettingKey {
			continue
		}
		resolved[key] = value
	}

	config, err := service.ResolveSMTPConfig(resolved)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	effectiveSettings := service.ResolveSettings(h.defaults, storedSettings)
	if err := h.emailSvc.SendTestEmail(c.Request.Context(), *config, user.Email, effectiveSettings[service.SiteNameSettingKey]); err != nil {
		Fail(c, 502, 50200, "测试邮件发送失败："+err.Error())
		return
	}

	Success(c, gin.H{"sent_to": user.Email})
}
