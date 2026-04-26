package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
)

// templateTranslator returns the closure templates use as `{{call .T "key"}}`.
// Closures (instead of template.FuncMap) are how we get a per-request locale
// into the template engine: html/template's FuncMap is resolved at parse
// time, so the only way to thread a per-request value through is to bind
// it into a value the template invokes via `{{call ...}}`.
//
// Variadic args support format strings — `{{call .T "upload.subtitle"
// .UploadMaxFileSizeLabel}}` resolves the catalogue entry then sprintf's
// the labels in. Templates that don't pass args still work because [i18n.T]
// short-circuits to the literal entry when no args are supplied.
func templateTranslator(c *gin.Context) func(string, ...any) string {
	locale := middleware.LocaleFromGin(c)
	return func(key string, args ...any) string {
		return i18n.T(locale, key, args...)
	}
}

func loadResolvedSettings(ctx context.Context, settingRepo *repo.SettingRepo, defaults map[string]string) map[string]string {
	overrides, err := settingRepo.GetAll(ctx)
	if err != nil {
		return service.ResolveSettings(defaults, nil)
	}
	return service.ResolveSettings(defaults, overrides)
}

// landingTemplateData assembles the per-request data map every public-page
// template reads. The gin context is now required because we inject a
// per-request translator (`T`) and the active locale (`Lang`) so templates
// can render strings without hard-coding any one language.
func landingTemplateData(c *gin.Context, currentUser *publicUser, settings map[string]string, activeNav, pageTitle string) gin.H {
	locale := middleware.LocaleFromGin(c)
	return gin.H{
		"CurrentUser":         currentUser,
		"ActiveNav":           activeNav,
		"DocumentTitle":       buildPageTitle(settings[service.SiteTitleSettingKey], pageTitle),
		"SiteName":            settings[service.SiteNameSettingKey],
		"SiteTitle":           settings[service.SiteTitleSettingKey],
		"SiteKeywords":        settings[service.SiteKeywordsSettingKey],
		"SiteDescription":     settings[service.SiteDescriptionSettingKey],
		"SiteFaviconURL":      settings[service.SiteFaviconURLSettingKey],
		"SiteHeaderBrand":     settings[service.SiteHeaderBrandSettingKey],
		"SiteHeaderGitHubURL": settings[service.SiteHeaderNavGitHubURLSettingKey],
		"SiteFooterText":      settings[service.SiteFooterTextSettingKey],
		"SiteFooterCopyright": settings[service.SiteFooterCopyrightSettingKey],
		// Translation surface — every template that renders text reaches
		// for these. Lang feeds <html lang="…"> / hreflang signals;
		// SupportedLocales drives the language switcher dropdown.
		"T":                templateTranslator(c),
		"Lang":             string(locale),
		"SupportedLocales": i18n.SupportedLocales(),
		"LocaleLabels":     i18n.LocaleLabels,
	}
}

func buildPageTitle(siteTitle, pageTitle string) string {
	siteTitle = strings.TrimSpace(siteTitle)
	pageTitle = strings.TrimSpace(pageTitle)
	if siteTitle == "" {
		siteTitle = "Kite"
	}
	if pageTitle == "" || pageTitle == siteTitle {
		return siteTitle
	}
	return fmt.Sprintf("%s - %s", pageTitle, siteTitle)
}

func buildAdminPageTitle(c *gin.Context, settings map[string]string) string {
	brand := strings.TrimSpace(settings[service.SiteNameSettingKey])
	if brand == "" {
		brand = strings.TrimSpace(settings[service.SiteHeaderBrandSettingKey])
	}
	if brand == "" {
		brand = "Kite"
	}
	locale := middleware.LocaleFromGin(c)
	return i18n.T(locale, "admin.page_title_suffix", brand)
}

func resolveUploadMaxFileSizeBytes(settings map[string]string, fallback int64) int64 {
	if raw, ok := settings[service.UploadMaxFileSizeMBSettingKey]; ok {
		if parsed, err := service.ParseUploadMaxFileSizeBytes(raw); err == nil {
			return parsed
		}
	}
	return fallback
}

func formatUploadMaxFileSizeLabel(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}
