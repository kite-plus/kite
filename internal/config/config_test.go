package config

import (
	"testing"
	"time"
)

func TestDefaultConfig_SiteDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Site.Name != "Kite" {
		t.Errorf("Site.Name: want %q, got %q", "Kite", cfg.Site.Name)
	}
	if cfg.Site.URL != "http://localhost:8080" {
		t.Errorf("Site.URL: want %q, got %q", "http://localhost:8080", cfg.Site.URL)
	}
}

func TestDefaultConfig_UploadDefaults(t *testing.T) {
	cfg := DefaultConfig()

	const wantMaxSize = 100 * 1024 * 1024
	if cfg.Upload.MaxFileSize != wantMaxSize {
		t.Errorf("MaxFileSize: want %d, got %d", wantMaxSize, cfg.Upload.MaxFileSize)
	}
	if cfg.Upload.ThumbWidth != 300 {
		t.Errorf("ThumbWidth: want 300, got %d", cfg.Upload.ThumbWidth)
	}
	if cfg.Upload.ThumbQuality != 80 {
		t.Errorf("ThumbQuality: want 80, got %d", cfg.Upload.ThumbQuality)
	}
	if cfg.Upload.PathPattern != "{year}/{month}/{md5_8}/{uuid}.{ext}" {
		t.Errorf("PathPattern: %q", cfg.Upload.PathPattern)
	}
	if cfg.Upload.AutoWebP {
		t.Error("AutoWebP should default to false")
	}
	if cfg.Upload.AllowDuplicate {
		t.Error("AllowDuplicate should default to false")
	}
	if len(cfg.Upload.ForbiddenExts) == 0 {
		t.Error("ForbiddenExts should not be empty by default")
	}
}

func TestDefaultConfig_AuthDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Auth.AccessTokenExpiry != 2*time.Hour {
		t.Errorf("AccessTokenExpiry: want 2h, got %v", cfg.Auth.AccessTokenExpiry)
	}
	if cfg.Auth.RefreshTokenExpiry != 7*24*time.Hour {
		t.Errorf("RefreshTokenExpiry: want 168h, got %v", cfg.Auth.RefreshTokenExpiry)
	}
	if !cfg.Auth.AllowRegistration {
		t.Error("AllowRegistration should default to true")
	}
}

func TestDefaultConfig_ServerDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host: want %q, got %q", "0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port: want 8080, got %d", cfg.Server.Port)
	}
}

func TestDefaultConfig_DatabaseDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Database.Driver: want %q, got %q", "sqlite", cfg.Database.Driver)
	}
	if cfg.Database.DSN == "" {
		t.Error("Database.DSN should not be empty")
	}
}
