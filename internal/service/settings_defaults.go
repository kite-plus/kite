package service

import (
	"fmt"
	"strings"
	"time"
)

const (
	SiteNameSettingKey               = "site_name"
	SiteURLSettingKey                = "site_url"
	AllowRegistrationSettingKey      = "allow_registration"
	AllowGuestUploadSettingKey       = "allow_guest_upload"
	AllowPublicGallerySettingKey     = "allow_public_gallery"
	SiteTitleSettingKey              = "site_title"
	SiteKeywordsSettingKey           = "site_keywords"
	SiteDescriptionSettingKey        = "site_description"
	SiteFaviconURLSettingKey         = "site_favicon_url"
	SiteHeaderBrandSettingKey        = "site_header_brand"
	SiteHeaderNavGitHubURLSettingKey = "site_header_nav_github_url"
	SiteFooterTextSettingKey         = "site_footer_text"
	SiteFooterCopyrightSettingKey    = "site_footer_copyright"
)

const (
	defaultSiteName       = "Kite"
	defaultSiteKeywords   = "图床,媒体托管,自部署,图片托管,S3,PicGo,开源"
	defaultSiteFaviconURL = "/favicon.svg"
	defaultSiteGitHubURL  = "https://github.com/amigoer/kite"
	defaultSiteFooterText = "开源自部署媒体托管系统"
)

// DefaultSettings returns the runtime-default settings used when the database
// has not stored an override yet.
func DefaultSettings(siteName, siteURL string, allowRegistration bool, uploadPathPattern string, uploadMaxFileSize int64) map[string]string {
	name := strings.TrimSpace(siteName)
	if name == "" {
		name = defaultSiteName
	}

	return map[string]string{
		SiteNameSettingKey:               name,
		SiteURLSettingKey:                strings.TrimSpace(siteURL),
		AllowRegistrationSettingKey:      boolString(allowRegistration),
		AllowGuestUploadSettingKey:       "false",
		AllowPublicGallerySettingKey:     "false",
		UploadPathPatternSettingKey:      strings.TrimSpace(uploadPathPattern),
		UploadMaxFileSizeMBSettingKey:    DefaultUploadMaxFileSizeMB(uploadMaxFileSize),
		SiteKeywordsSettingKey:           defaultSiteKeywords,
		SiteFaviconURLSettingKey:         defaultSiteFaviconURL,
		SiteHeaderNavGitHubURLSettingKey: defaultSiteGitHubURL,
		SiteFooterTextSettingKey:         defaultSiteFooterText,
	}
}

// ResolveSettings merges stored settings with runtime defaults while also
// filling derived website display values such as site_title and footer
// copyright.
func ResolveSettings(defaults, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(defaults)+len(overrides)+4)
	for key, value := range defaults {
		merged[key] = strings.TrimSpace(value)
	}
	for key, value := range overrides {
		merged[key] = strings.TrimSpace(value)
	}

	siteName := resolveSettingValue(defaults, overrides, SiteNameSettingKey, firstNonEmptySetting(defaults[SiteNameSettingKey], defaultSiteName), false)
	merged[SiteNameSettingKey] = siteName
	merged[SiteURLSettingKey] = resolveSettingValue(defaults, overrides, SiteURLSettingKey, strings.TrimSpace(defaults[SiteURLSettingKey]), false)
	merged[AllowRegistrationSettingKey] = resolveSettingValue(defaults, overrides, AllowRegistrationSettingKey, strings.TrimSpace(defaults[AllowRegistrationSettingKey]), false)
	merged[AllowGuestUploadSettingKey] = resolveSettingValue(defaults, overrides, AllowGuestUploadSettingKey, strings.TrimSpace(defaults[AllowGuestUploadSettingKey]), false)
	merged[AllowPublicGallerySettingKey] = resolveSettingValue(defaults, overrides, AllowPublicGallerySettingKey, strings.TrimSpace(defaults[AllowPublicGallerySettingKey]), false)
	merged[UploadPathPatternSettingKey] = resolveSettingValue(defaults, overrides, UploadPathPatternSettingKey, strings.TrimSpace(defaults[UploadPathPatternSettingKey]), false)
	merged[UploadMaxFileSizeMBSettingKey] = resolveSettingValue(defaults, overrides, UploadMaxFileSizeMBSettingKey, strings.TrimSpace(defaults[UploadMaxFileSizeMBSettingKey]), false)

	merged[SiteTitleSettingKey] = resolveSettingValue(defaults, overrides, SiteTitleSettingKey, fmt.Sprintf("%s - 自部署媒体托管系统", siteName), false)
	merged[SiteKeywordsSettingKey] = resolveSettingValue(defaults, overrides, SiteKeywordsSettingKey, defaultSiteKeywords, true)
	merged[SiteDescriptionSettingKey] = resolveSettingValue(defaults, overrides, SiteDescriptionSettingKey, fmt.Sprintf("%s 是一个开源的自部署媒体托管系统，支持图片、视频、音频和文件的上传、管理与分享。", siteName), true)
	merged[SiteFaviconURLSettingKey] = resolveSettingValue(defaults, overrides, SiteFaviconURLSettingKey, defaultSiteFaviconURL, false)
	merged[SiteHeaderBrandSettingKey] = resolveSettingValue(defaults, overrides, SiteHeaderBrandSettingKey, siteName, false)
	merged[SiteHeaderNavGitHubURLSettingKey] = resolveSettingValue(defaults, overrides, SiteHeaderNavGitHubURLSettingKey, defaultSiteGitHubURL, true)
	merged[SiteFooterTextSettingKey] = resolveSettingValue(defaults, overrides, SiteFooterTextSettingKey, defaultSiteFooterText, true)
	merged[SiteFooterCopyrightSettingKey] = resolveSettingValue(defaults, overrides, SiteFooterCopyrightSettingKey, fmt.Sprintf("© %d %s", time.Now().Year(), siteName), true)

	return merged
}

func resolveSettingValue(defaults, overrides map[string]string, key, fallback string, allowEmpty bool) string {
	if value, ok := overrides[key]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" || allowEmpty {
			return trimmed
		}
	}
	if value, ok := defaults[key]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" || allowEmpty {
			return trimmed
		}
	}
	return strings.TrimSpace(fallback)
}

func firstNonEmptySetting(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
