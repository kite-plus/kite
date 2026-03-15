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

	return nil
}
