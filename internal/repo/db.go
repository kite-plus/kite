package repo

import (
	"fmt"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/model"
	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	var dialector gorm.Dialector
	switch cfg.Database.Driver {
	case config.DatabaseDriverSQLite:
		dialector = sqlite.Open(cfg.Database.Path)
	case config.DatabaseDriverPostgres:
		dialector = postgres.Open(buildPostgresDSN(cfg.Database))
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("auto migrate database: %w", err)
	}

	return db, nil
}

// InitSQLiteDB 直接用 SQLite 路径初始化数据库（安装引导用）
func InitSQLiteDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.AdminSession{},
		&model.Tag{},
		&model.Category{},
		&model.Post{},
		&model.FriendLink{},
		&model.Comment{},
		&model.Page{},
		&model.Setting{},
	)
}

func buildPostgresDSN(cfg config.DatabaseConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)
}
