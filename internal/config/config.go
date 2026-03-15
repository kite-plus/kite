package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

const (
	RenderModeClassic      = "classic"
	RenderModeHeadless     = "headless"
	DatabaseDriverSQLite   = "sqlite"
	DatabaseDriverPostgres = "postgres"
)

type Config struct {
	RenderMode string         `yaml:"render_mode"`
	Database   DatabaseConfig `yaml:"database"`
	Admin      AdminConfig    `yaml:"admin"`
}

type DatabaseConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
	Path     string `yaml:"path"`
}

type AdminConfig struct {
	Enabled         bool               `yaml:"enabled"`
	Username        string             `yaml:"username"`
	PasswordHash    string             `yaml:"password_hash"`
	SessionSecret   string             `yaml:"session_secret"`
	SessionTTLHours int                `yaml:"session_ttl_hours"`
	Profile         AdminProfileConfig `yaml:"profile"`
}

type AdminProfileConfig struct {
	DisplayName string `yaml:"display_name"`
	Email       string `yaml:"email"`
	Bio         string `yaml:"bio"`
	Avatar      string `yaml:"avatar"`
	Website     string `yaml:"website"`
	Location    string `yaml:"location"`
}

func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config yaml: %w", err)
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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
