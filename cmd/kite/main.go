package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/amigoer/kite/internal/api"
	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/amigoer/kite/internal/storage"
	"github.com/amigoer/kite/web"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func main() {
	cfg := config.DefaultConfig()

	// 环境变量覆盖
	if port := os.Getenv("KITE_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Server.Port)
	}
	if host := os.Getenv("KITE_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if dsn := os.Getenv("KITE_DSN"); dsn != "" {
		cfg.Database.DSN = dsn
	}
	if driver := os.Getenv("KITE_DB_DRIVER"); driver != "" {
		cfg.Database.Driver = driver
	}
	if siteURL := os.Getenv("KITE_SITE_URL"); siteURL != "" {
		cfg.Site.URL = siteURL
	}

	// 确保数据目录存在
	dataDir := filepath.Dir(cfg.Database.DSN)
	if dataDir != "." {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("failed to create data directory: %v", err)
		}
	}

	// 初始化数据库
	db, err := initDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}

	// 自动迁移表结构
	if err := autoMigrate(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 从数据库加载运行时配置
	settingRepo := repo.NewSettingRepo(db)
	loadRuntimeConfig(settingRepo, &cfg)

	// 将存量绝对 URL 迁移为相对路径
	migrateAbsoluteURLs(db)

	// 确保 JWT 密钥存在（首次启动自动生成并持久化）
	if err := ensureJWTSecret(settingRepo, &cfg); err != nil {
		log.Fatalf("failed to ensure jwt secret: %v", err)
	}

	// 初始化存储管理器
	storageMgr := storage.NewManager()
	storageRepo := repo.NewStorageConfigRepo(db)
	seedDefaultStorage(storageRepo, dataDir)
	loadStorageConfigs(storageRepo, storageMgr)

	// 初始化服务
	userRepo := repo.NewUserRepo(db)
	tokenRepo := repo.NewAPITokenRepo(db)
	fileRepo := repo.NewFileRepo(db)

	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg.Auth)

	// 首次启动：无用户时自动创建默认管理员
	seedDefaultAdmin(userRepo, authSvc)

	imageSvc := service.NewImageService(cfg.Upload.ThumbWidth, cfg.Upload.ThumbQuality)
	fileSvc := service.NewFileService(fileRepo, userRepo, storageMgr, imageSvc, cfg.Upload)

	// 加载内嵌资产
	var adminFS fs.FS
	if sub, err := fs.Sub(web.AdminFS, "admin/dist"); err == nil {
		adminFS = sub
	}
	var templateFS fs.FS
	if sub, err := fs.Sub(web.AdminFS, "template"); err == nil {
		templateFS = sub
	}

	// 设置路由
	router := api.SetupRouter(api.RouterConfig{
		DB:         db,
		StorageMgr: storageMgr,
		AuthSvc:    authSvc,
		FileSvc:    fileSvc,
		AdminFS:    adminFS,
		TemplateFS: templateFS,
	})

	// 启动 HTTP 服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 优雅关闭
	go func() {
		log.Printf("Kite server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	log.Println("server stopped")
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.File{},
		&model.Album{},
		&model.StorageConfig{},
		&model.APIToken{},
		&model.UploadSession{},
		&model.Setting{},
	)
}

// loadRuntimeConfig 从数据库 settings 表加载运行时配置。
func loadRuntimeConfig(settingRepo *repo.SettingRepo, cfg *config.Config) {
	ctx := context.Background()
	settings, err := settingRepo.GetAll(ctx)
	if err != nil {
		return
	}

	if v, ok := settings["site_name"]; ok {
		cfg.Site.Name = v
	}
	if v, ok := settings["site_url"]; ok {
		cfg.Site.URL = v
	}
	if v, ok := settings["jwt_secret"]; ok {
		cfg.Auth.JWTSecret = v
	}
	if _, ok := settings["allow_registration"]; ok {
		cfg.Auth.AllowRegistration = settings["allow_registration"] == "true"
	}
}

// ensureJWTSecret 保证 JWT 签名密钥存在。
// 若 settings 表中未配置，则生成 32 字节随机密钥并持久化，避免空密钥导致 token 可伪造。
func ensureJWTSecret(settingRepo *repo.SettingRepo, cfg *config.Config) error {
	if cfg.Auth.JWTSecret != "" {
		return nil
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generate jwt secret: %w", err)
	}
	secret := hex.EncodeToString(b)
	if err := settingRepo.Set(context.Background(), "jwt_secret", secret); err != nil {
		return fmt.Errorf("persist jwt secret: %w", err)
	}
	cfg.Auth.JWTSecret = secret
	log.Println("generated new JWT secret and saved to settings")
	return nil
}

// seedDefaultAdmin 首次启动时自动创建默认管理员账号 admin/admin。
// 该账号标记为 PasswordMustChange，首次登录后强制在前端重置用户名与密码才可使用。
func seedDefaultAdmin(userRepo *repo.UserRepo, authSvc *service.AuthService) {
	ctx := context.Background()
	count, err := userRepo.Count(ctx)
	if err != nil || count > 0 {
		return
	}

	if _, err := authSvc.CreateAdminUser(ctx, "admin", "admin@kite.local", "admin", true); err != nil {
		log.Fatalf("failed to create default admin: %v", err)
	}

	log.Println("============================================================")
	log.Println("  ⚠️  Default admin account created with WEAK credentials:")
	log.Println("        Username: admin")
	log.Println("        Password: admin")
	log.Println("  You MUST change the username and password on first login.")
	log.Println("  Do NOT expose this server to the public internet until you")
	log.Println("  have completed the first-login reset.")
	log.Println("============================================================")
}

// migrateAbsoluteURLs 将存量记录中的绝对 URL（如 http://localhost:8080/i/xxx）改写为相对路径（/i/xxx）。
// 相对路径在响应时由 handler 根据请求 Host 动态拼接，不再依赖启动时配置的 site_url。
func migrateAbsoluteURLs(db *gorm.DB) {
	var count int64
	var files []model.File
	if err := db.Where("url LIKE ?", "http%").Find(&files).Error; err != nil {
		return
	}
	for _, f := range files {
		for _, prefix := range []string{"/i/", "/v/", "/a/", "/f/"} {
			if idx := strings.Index(f.URL, prefix); idx >= 0 {
				_ = db.Model(&model.File{}).Where("id = ?", f.ID).Update("url", f.URL[idx:]).Error
				count++
				break
			}
		}
	}
	var thumbFiles []model.File
	if err := db.Where("thumb_url IS NOT NULL AND thumb_url LIKE ?", "http%").Find(&thumbFiles).Error; err == nil {
		for _, f := range thumbFiles {
			if f.ThumbURL == nil {
				continue
			}
			if idx := strings.Index(*f.ThumbURL, "/t/"); idx >= 0 {
				newURL := (*f.ThumbURL)[idx:]
				_ = db.Model(&model.File{}).Where("id = ?", f.ID).Update("thumb_url", newURL).Error
				count++
			} else if len(f.HashMD5) >= 8 {
				newURL := "/t/" + f.HashMD5[:8]
				_ = db.Model(&model.File{}).Where("id = ?", f.ID).Update("thumb_url", newURL).Error
				count++
			}
		}
	}
	if count > 0 {
		log.Printf("migrated %d URLs from absolute to relative paths", count)
	}
}

// seedDefaultStorage 首次启动时自动创建一个本地存储作为兜底，避免用户必须先手动添加存储才能使用上传功能。
// 仅在数据库中不存在任何存储配置时才创建；用户手动删除后重启不会被再次创建。
func seedDefaultStorage(storageRepo *repo.StorageConfigRepo, dataDir string) {
	ctx := context.Background()
	existing, err := storageRepo.List(ctx)
	if err != nil {
		log.Printf("warning: failed to check existing storage configs: %v", err)
		return
	}
	if len(existing) > 0 {
		return
	}

	basePath := filepath.Join(dataDir, "uploads")
	lc := storage.LocalConfig{BasePath: basePath}
	raw, err := json.Marshal(lc)
	if err != nil {
		log.Printf("warning: failed to marshal default storage config: %v", err)
		return
	}

	cfg := &model.StorageConfig{
		ID:        uuid.New().String(),
		Name:      "本机存储",
		Driver:    "local",
		Config:    string(raw),
		IsDefault: true,
		IsActive:  true,
	}
	if err := storageRepo.Create(ctx, cfg); err != nil {
		log.Printf("warning: failed to seed default storage: %v", err)
		return
	}
	log.Printf("seeded default local storage at %s", basePath)
}

// loadStorageConfigs 从数据库加载所有活跃的存储配置到管理器。
func loadStorageConfigs(storageRepo *repo.StorageConfigRepo, mgr *storage.Manager) {
	configs, err := storageRepo.ListActive(context.Background())
	if err != nil {
		log.Printf("warning: failed to load storage configs: %v", err)
		return
	}

	for _, cfg := range configs {
		var scfg storage.StorageConfig
		scfg.Driver = cfg.Driver

		switch cfg.Driver {
		case "local":
			var lc storage.LocalConfig
			if err := json.Unmarshal([]byte(cfg.Config), &lc); err != nil {
				log.Printf("warning: failed to parse local config %s: %v", cfg.ID, err)
				continue
			}
			scfg.Local = &lc
		case "s3":
			var sc storage.S3Config
			if err := json.Unmarshal([]byte(cfg.Config), &sc); err != nil {
				log.Printf("warning: failed to parse s3 config %s: %v", cfg.ID, err)
				continue
			}
			scfg.S3 = &sc
		}

		if err := mgr.LoadAndRegister(cfg.ID, scfg); err != nil {
			log.Printf("warning: failed to load storage %s (%s): %v", cfg.Name, cfg.ID, err)
			continue
		}

		if cfg.IsDefault {
			if err := mgr.SetDefault(cfg.ID); err != nil {
				log.Printf("warning: failed to set default storage: %v", err)
			}
		}

		log.Printf("loaded storage: %s (%s, %s)", cfg.Name, cfg.Driver, cfg.ID)
	}
}
