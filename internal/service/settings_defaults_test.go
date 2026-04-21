package service

import (
	"fmt"
	"testing"
	"time"
)

var settingsDefaultForbiddenExts = []string{".exe", ".bat", ".cmd", ".sh", ".ps1"}

func TestResolveSettingsDerivesSiteDisplayDefaults(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)
	overrides := map[string]string{
		SiteNameSettingKey: "媒体仓库",
	}

	got := ResolveSettings(defaults, overrides)

	if got[SiteTitleSettingKey] != "媒体仓库 - 自部署媒体托管系统" {
		t.Fatalf("unexpected site_title: %q", got[SiteTitleSettingKey])
	}
	if got[SiteHeaderBrandSettingKey] != "媒体仓库" {
		t.Fatalf("unexpected site_header_brand: %q", got[SiteHeaderBrandSettingKey])
	}
	if got[SiteFaviconURLSettingKey] != "/favicon.svg" {
		t.Fatalf("unexpected site_favicon_url: %q", got[SiteFaviconURLSettingKey])
	}
	wantCopyright := fmt.Sprintf("© %d 媒体仓库", time.Now().Year())
	if got[SiteFooterCopyrightSettingKey] != wantCopyright {
		t.Fatalf("unexpected site_footer_copyright: %q", got[SiteFooterCopyrightSettingKey])
	}
}

func TestResolveSettingsKeepsBlankOptionalDisplayFields(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)
	overrides := map[string]string{
		SiteHeaderNavGitHubURLSettingKey: "",
		SiteFooterTextSettingKey:         "",
	}

	got := ResolveSettings(defaults, overrides)

	if got[SiteHeaderNavGitHubURLSettingKey] != "" {
		t.Fatalf("expected blank github url, got %q", got[SiteHeaderNavGitHubURLSettingKey])
	}
	if got[SiteFooterTextSettingKey] != "" {
		t.Fatalf("expected blank footer text, got %q", got[SiteFooterTextSettingKey])
	}
}

func TestResolveSettingsFallsBackToDefaultFaviconWhenBlank(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)
	overrides := map[string]string{
		SiteFaviconURLSettingKey: "   ",
	}

	got := ResolveSettings(defaults, overrides)

	if got[SiteFaviconURLSettingKey] != "/favicon.svg" {
		t.Fatalf("expected default favicon url, got %q", got[SiteFaviconURLSettingKey])
	}
}

func TestResolveSettingsIncludesUploadMaxFileSize(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)

	got := ResolveSettings(defaults, map[string]string{
		UploadMaxFileSizeMBSettingKey: "256",
	})

	if got[UploadMaxFileSizeMBSettingKey] != "256" {
		t.Fatalf("unexpected upload.max_file_size_mb: %q", got[UploadMaxFileSizeMBSettingKey])
	}
}

func TestResolveSettingsIncludesDefaultQuota(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)

	got := ResolveSettings(defaults, map[string]string{
		DefaultQuotaSettingKey: "10 GB",
	})

	if got[DefaultQuotaSettingKey] != DefaultQuotaSettingValue() {
		t.Fatalf("unexpected default_quota: %q", got[DefaultQuotaSettingKey])
	}
}

func TestResolveSettingsIncludesRateLimitSettings(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)

	got := ResolveSettings(defaults, map[string]string{
		AuthRateLimitPerMinuteSettingKey:        "30",
		GuestUploadRateLimitPerMinuteSettingKey: "120",
	})

	if got[AuthRateLimitPerMinuteSettingKey] != "30" {
		t.Fatalf("unexpected auth rate limit: %q", got[AuthRateLimitPerMinuteSettingKey])
	}
	if got[GuestUploadRateLimitPerMinuteSettingKey] != "120" {
		t.Fatalf("unexpected guest upload rate limit: %q", got[GuestUploadRateLimitPerMinuteSettingKey])
	}
}

func TestResolveSettingsIncludesSMTPDefaults(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)

	got := ResolveSettings(defaults, nil)

	if got[SMTPPortSettingKey] != "587" {
		t.Fatalf("unexpected smtp_port: %q", got[SMTPPortSettingKey])
	}
	if got[SMTPTLSSettingKey] != "false" {
		t.Fatalf("unexpected smtp_tls: %q", got[SMTPTLSSettingKey])
	}
}

func TestResolveSettingsIncludesDangerousExtensionDefaults(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024, settingsDefaultForbiddenExts)

	got := ResolveSettings(defaults, nil)

	if got[UploadDangerousExtensionRulesSettingKey] != `[{"ext":".exe","action":"block"},{"ext":".bat","action":"block"},{"ext":".cmd","action":"block"},{"ext":".sh","action":"block"},{"ext":".ps1","action":"block"}]` {
		t.Fatalf("unexpected upload.dangerous_extension_rules: %q", got[UploadDangerousExtensionRulesSettingKey])
	}
	if got[UploadDangerousRenameSuffixSettingKey] != DefaultDangerousRenameSuffixValue {
		t.Fatalf("unexpected upload.dangerous_rename_suffix: %q", got[UploadDangerousRenameSuffixSettingKey])
	}
}
