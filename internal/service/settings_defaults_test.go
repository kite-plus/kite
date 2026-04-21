package service

import (
	"fmt"
	"testing"
	"time"
)

func TestResolveSettingsDerivesSiteDisplayDefaults(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024)
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
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024)
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
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024)
	overrides := map[string]string{
		SiteFaviconURLSettingKey: "   ",
	}

	got := ResolveSettings(defaults, overrides)

	if got[SiteFaviconURLSettingKey] != "/favicon.svg" {
		t.Fatalf("expected default favicon url, got %q", got[SiteFaviconURLSettingKey])
	}
}

func TestResolveSettingsIncludesUploadMaxFileSize(t *testing.T) {
	defaults := DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024)

	got := ResolveSettings(defaults, map[string]string{
		UploadMaxFileSizeMBSettingKey: "256",
	})

	if got[UploadMaxFileSizeMBSettingKey] != "256" {
		t.Fatalf("unexpected upload.max_file_size_mb: %q", got[UploadMaxFileSizeMBSettingKey])
	}
}
