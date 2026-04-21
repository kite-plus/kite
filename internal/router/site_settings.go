package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

func loadResolvedSettings(ctx context.Context, settingRepo *repo.SettingRepo, defaults map[string]string) map[string]string {
	overrides, err := settingRepo.GetAll(ctx)
	if err != nil {
		return service.ResolveSettings(defaults, nil)
	}
	return service.ResolveSettings(defaults, overrides)
}

func landingTemplateData(currentUser *publicUser, settings map[string]string, activeNav, pageTitle string) gin.H {
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

func buildAdminPageTitle(settings map[string]string) string {
	brand := strings.TrimSpace(settings[service.SiteNameSettingKey])
	if brand == "" {
		brand = strings.TrimSpace(settings[service.SiteHeaderBrandSettingKey])
	}
	if brand == "" {
		brand = "Kite"
	}
	return fmt.Sprintf("%s 管理后台", brand)
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
