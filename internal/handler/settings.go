package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
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
//
// Secrets (JWT signing key, SMTP password, per-provider OAuth client secrets)
// are stripped from the response and replaced with a "<key>_configured" flag
// so the admin UI can show a "Set"/"Not set" badge without ever seeing the
// plaintext. Anyone with read access to the settings API must never be able
// to exfiltrate the JWT secret — that would let them forge arbitrary tokens.
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.settingRepo.GetAll(c.Request.Context())
	if err != nil {
		ServerError(c, M(c, i18n.KeySettingsGetFailed))
		return
	}
	merged := service.ResolveSettings(h.defaults, settings)

	// Collect every secret key the operator might have configured: both the
	// ones the defaults seed (e.g. smtp_password) and the ones only present
	// once the operator writes them (jwt_secret, oauth_*_client_secret).
	secretKeys := map[string]struct{}{
		service.JWTSecretSettingKey:    {},
		service.SMTPPasswordSettingKey: {},
	}
	for key := range settings {
		if service.IsSecretSettingKey(key) {
			secretKeys[key] = struct{}{}
		}
	}
	for key := range merged {
		if service.IsSecretSettingKey(key) {
			secretKeys[key] = struct{}{}
		}
	}

	for key := range secretKeys {
		configured := "false"
		if strings.TrimSpace(settings[key]) != "" {
			configured = "true"
		}
		merged[service.SecretConfiguredKey(key)] = configured
		delete(merged, key)
	}
	Success(c, merged)
}

type updateSettingsRequest struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

// Update bulk-updates system settings.
func (h *SettingsHandler) Update(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeySettingsInvalidData, err.Error()))
		return
	}

	// Reject writes to read-only secrets (currently only jwt_secret). Rotating
	// the JWT key via this endpoint would both invalidate every active session
	// and let an admin ship tokens signed with an attacker-controlled value.
	for key := range req.Settings {
		if service.IsReadOnlySecretSettingKey(key) {
			BadRequest(c, M(c, i18n.KeySettingsCannotModify, key))
			return
		}
	}
	// Silently drop every "*_configured" sentinel — those are GET-only flags
	// emitted by Get(); accepting them would overwrite the real secret key
	// with the literal string "true"/"false".
	for key := range req.Settings {
		if service.IsSecretConfiguredKey(key) {
			delete(req.Settings, key)
		}
	}

	if raw, ok := req.Settings[service.UploadPathPatternSettingKey]; ok {
		pattern, err := service.NormalizeUploadPathPattern(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidPathPattern, err.Error()))
			return
		}
		req.Settings[service.UploadPathPatternSettingKey] = strings.TrimSpace(pattern)
	}
	if raw, ok := req.Settings[service.DefaultQuotaSettingKey]; ok {
		normalized, err := service.NormalizeDefaultQuota(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidValue, service.DefaultQuotaSettingKey, err.Error()))
			return
		}
		req.Settings[service.DefaultQuotaSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.UploadMaxFileSizeMBSettingKey]; ok {
		normalized, err := service.NormalizeUploadMaxFileSizeMB(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidMaxFileSize, err.Error()))
			return
		}
		req.Settings[service.UploadMaxFileSizeMBSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.UploadDangerousExtensionRulesSettingKey]; ok {
		normalized, err := service.NormalizeDangerousExtensionRules(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidDangerousExtension, err.Error()))
			return
		}
		req.Settings[service.UploadDangerousExtensionRulesSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.UploadDangerousRenameSuffixSettingKey]; ok {
		normalized, err := service.NormalizeDangerousRenameSuffix(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidDangerousRename, err.Error()))
			return
		}
		req.Settings[service.UploadDangerousRenameSuffixSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPPortSettingKey]; ok {
		normalized, err := service.NormalizeSMTPPort(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidValue, service.SMTPPortSettingKey, err.Error()))
			return
		}
		req.Settings[service.SMTPPortSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPTLSSettingKey]; ok {
		normalized, err := service.NormalizeSMTPBool(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidValue, service.SMTPTLSSettingKey, err.Error()))
			return
		}
		req.Settings[service.SMTPTLSSettingKey] = normalized
	}
	if raw, ok := req.Settings[service.SMTPFromSettingKey]; ok {
		normalized, err := service.NormalizeSMTPFrom(raw)
		if err != nil {
			BadRequest(c, M(c, i18n.KeySettingsInvalidValue, service.SMTPFromSettingKey, err.Error()))
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
			BadRequest(c, M(c, i18n.KeySettingsInvalidValue, key, err.Error()))
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
			BadRequest(c, M(c, i18n.KeySettingsInvalidEmpty, key))
			return
		}
		req.Settings[key] = trimmed
	}

	if err := h.settingRepo.SetBatch(c.Request.Context(), req.Settings); err != nil {
		ServerError(c, M(c, i18n.KeySettingsUpdateFailed))
		return
	}

	Success(c, nil)
}

func (h *SettingsHandler) TestEmail(c *gin.Context) {
	if h.emailSvc == nil || h.userRepo == nil {
		ServerError(c, M(c, i18n.KeyEmailServiceNotConfigured))
		return
	}

	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeySettingsInvalidData, err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Unauthorized(c, M(c, i18n.KeyAuthUserNotFound))
			return
		}
		ServerError(c, M(c, i18n.KeyAuthLoadCurrentUserFailed))
		return
	}

	storedSettings, err := h.settingRepo.GetAll(c.Request.Context())
	if err != nil {
		ServerError(c, M(c, i18n.KeySettingsGetFailed))
		return
	}

	resolved := make(map[string]string, len(storedSettings)+len(req.Settings))
	for key, value := range storedSettings {
		resolved[key] = value
	}
	for key, value := range req.Settings {
		// Drop GET-only sentinel flags — they aren't real SMTP fields.
		if service.IsSecretConfiguredKey(key) {
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
		Fail(c, 502, 50200, M(c, i18n.KeyEmailTestSendFailed, err.Error()))
		return
	}

	Success(c, gin.H{"sent_to": user.Email})
}
