package config

import (
	"fmt"
)

const (
	RenderModeClassic      = "classic"
	RenderModeHeadless     = "headless"
	DatabaseDriverSQLite   = "sqlite"
	DatabaseDriverPostgres = "postgres"
)

type Config struct {
	RenderMode string         `json:"render_mode"`
	Database   DatabaseConfig `json:"database"`
	Admin      AdminConfig    `json:"admin"`
	Site       SiteConfig     `json:"site"`
	Post       PostConfig     `json:"post"`
	AI         AIConfig       `json:"ai"`
}

type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SSLMode  string `json:"ssl_mode"`
	Path     string `json:"path"`
}

type AdminConfig struct {
	Enabled         bool               `json:"enabled"`
	Username        string             `json:"username"`
	PasswordHash    string             `json:"password_hash"`
	SessionSecret   string             `json:"session_secret"`
	SessionTTLHours int                `json:"session_ttl_hours"`
	Profile         AdminProfileConfig `json:"profile"`
}

type AdminProfileConfig struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
	Website     string `json:"website"`
	Location    string `json:"location"`
}

// SiteConfig 站点基础设置
type SiteConfig struct {
	SiteName    string `json:"site_name"`
	SiteURL     string `json:"site_url"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Favicon     string `json:"favicon"`
	Logo        string `json:"logo"`
	ICP         string `json:"icp"`
	Footer      string `json:"footer"`
}

// PostConfig 文章相关设置
type PostConfig struct {
	PostsPerPage    int    `json:"posts_per_page"`
	EnableComment   bool   `json:"enable_comment"`
	EnableToc       bool   `json:"enable_toc"`
	SummaryLength   int    `json:"summary_length"`
	DefaultCoverURL string `json:"default_cover_url"`
}

// AIConfig AI 集成设置
type AIConfig struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	AutoSummary bool   `json:"auto_summary"`
	AutoTag     bool   `json:"auto_tag"`
}


func Default() *Config {
	cfg := &Config{
		RenderMode: RenderModeClassic,
		Database: DatabaseConfig{
			Driver:  DatabaseDriverSQLite,
			Path:    "kite.db",
			Host:    "127.0.0.1",
			Port:    5432,
			SSLMode: "disable",
		},
		Admin: AdminConfig{
			SessionTTLHours: 168,
		},
		Site: SiteConfig{
			SiteName: "Kite",
		},
		Post: PostConfig{
			PostsPerPage:  10,
			EnableComment: true,
			EnableToc:     true,
			SummaryLength: 200,
		},
	}
	cfg.ApplyDefaults()
	return cfg
}

func (c *Config) ApplyDefaults() {
	if c.RenderMode == "" {
		c.RenderMode = RenderModeClassic
	}
	if c.Database.Driver == "" {
		c.Database.Driver = DatabaseDriverSQLite
	}
	if c.Database.Path == "" {
		c.Database.Path = "kite.db"
	}
	if c.Database.Host == "" {
		c.Database.Host = "127.0.0.1"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Admin.SessionTTLHours <= 0 {
		c.Admin.SessionTTLHours = 168
	}
	if c.Admin.Profile.DisplayName == "" && c.Admin.Username != "" {
		c.Admin.Profile.DisplayName = c.Admin.Username
	}
	if c.Site.SiteName == "" {
		c.Site.SiteName = "Kite"
	}
	if c.Post.PostsPerPage <= 0 {
		c.Post.PostsPerPage = 10
	}
	if c.Post.SummaryLength <= 0 {
		c.Post.SummaryLength = 200
	}
}

func (c *Config) Validate() error {
	switch c.RenderMode {
	case RenderModeClassic, RenderModeHeadless:
	default:
		return fmt.Errorf("invalid render_mode: %s", c.RenderMode)
	}

	switch c.Database.Driver {
	case DatabaseDriverSQLite:
		if c.Database.Path == "" {
			return fmt.Errorf("database.path is required when driver is sqlite")
		}
	case DatabaseDriverPostgres:
		if c.Database.Host == "" || c.Database.Port == 0 || c.Database.User == "" || c.Database.Name == "" {
			return fmt.Errorf("database host, port, user and name are required when driver is postgres")
		}
	default:
		return fmt.Errorf("unsupported database driver: %s", c.Database.Driver)
	}

	if c.Admin.Enabled {
		if c.Admin.Username == "" {
			return fmt.Errorf("admin.username is required when admin auth is enabled")
		}
		if c.Admin.PasswordHash == "" {
			return fmt.Errorf("admin.password_hash is required when admin auth is enabled")
		}
		if c.Admin.SessionSecret == "" {
			return fmt.Errorf("admin.session_secret is required when admin auth is enabled")
		}
		if c.Admin.SessionTTLHours <= 0 {
			return fmt.Errorf("admin.session_ttl_hours must be greater than 0 when admin auth is enabled")
		}
	}

	return nil
}
