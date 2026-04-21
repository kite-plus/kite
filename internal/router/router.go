// Package router builds the gin.Engine by wiring middleware, instantiating
// handlers from injected dependencies and delegating route registration to
// per-domain files (auth, file, album, token, user, storage, settings,
// system_status, public, landing, static).
//
// A single [Setup] call constructs every route. No handler or repository is
// created outside this package.
package router

import (
	"io/fs"
	"time"

	"github.com/amigoer/kite/internal/config"
	"github.com/amigoer/kite/internal/handler"
	"github.com/amigoer/kite/internal/middleware"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/amigoer/kite/internal/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Config carries all dependencies required to build the HTTP server. It is
// populated by the cmd entry point before calling [Setup].
type Config struct {
	DB                  *gorm.DB
	StorageMgr          *storage.Manager
	AuthSvc             *service.AuthService
	FileSvc             *service.FileService
	AuthConfig          config.AuthConfig
	UploadPathPattern   string
	UploadMaxFileSize   int64
	UploadForbiddenExts []string
	SiteName            string
	SiteURL             string
	AllowRegistration   bool
	AdminFS             fs.FS  // Embedded SPA assets (web/admin/dist).
	TemplateFS          fs.FS  // Embedded Go templates (web/template).
	DataDir             string // Data directory backing the local storage driver.
	ReloadStorage       func() // Rebuilds the storage manager after CRUD on storage configs.
}

// Setup constructs a fully wired gin.Engine: it creates the realtime metrics
// collector, installs global middleware, instantiates handlers from cfg and
// registers every route through the per-domain helpers in this package.
func Setup(cfg Config) *gin.Engine {
	r := gin.New()

	realtimeCollector := handler.NewRealtimeSystemStatusCollector()
	r.Use(middleware.Recovery())
	r.Use(middleware.AccessLog())
	r.Use(middleware.CORS())
	r.Use(realtimeCollector.Middleware())

	userRepo := repo.NewUserRepo(cfg.DB)
	fileRepo := repo.NewFileRepo(cfg.DB)
	albumRepo := repo.NewAlbumRepo(cfg.DB)
	tokenRepo := repo.NewAPITokenRepo(cfg.DB)
	identityRepo := repo.NewUserIdentityRepo(cfg.DB)
	storageRepo := repo.NewStorageConfigRepo(cfg.DB)
	settingRepo := repo.NewSettingRepo(cfg.DB)
	accessLogRepo := repo.NewFileAccessLogRepo(cfg.DB)
	settingDefaults := service.DefaultSettings(
		cfg.SiteName,
		cfg.SiteURL,
		cfg.AllowRegistration,
		cfg.UploadPathPattern,
		cfg.UploadMaxFileSize,
		cfg.UploadForbiddenExts,
	)

	oauthConfigSvc := service.NewOAuthConfigService(settingRepo, cfg.SiteURL)
	socialAuthSvc := service.NewSocialAuthService(
		cfg.AuthSvc,
		userRepo,
		identityRepo,
		settingRepo,
		oauthConfigSvc,
		cfg.AuthConfig.JWTSecret,
		cfg.AllowRegistration,
	)

	authHandler := handler.NewAuthHandler(cfg.AuthSvc, socialAuthSvc, oauthConfigSvc, userRepo, settingRepo, cfg.AllowRegistration, cfg.UploadMaxFileSize)
	fileHandler := handler.NewFileHandler(cfg.FileSvc, fileRepo, albumRepo, accessLogRepo)
	albumHandler := handler.NewAlbumHandler(albumRepo, fileRepo)
	tokenHandler := handler.NewTokenHandler(cfg.AuthSvc, tokenRepo)
	oauthProviderAdminHandler := handler.NewOAuthProviderAdminHandler(oauthConfigSvc)
	storageHandler := handler.NewStorageHandler(storageRepo, fileRepo, cfg.StorageMgr, cfg.ReloadStorage)
	emailSvc := service.NewEmailService()
	settingsHandler := handler.NewSettingsHandler(settingRepo, userRepo, emailSvc, settingDefaults)
	userHandler := handler.NewUserHandler(userRepo, fileRepo, accessLogRepo, cfg.AuthSvc)
	setupHandler := handler.NewSetupHandler(userRepo, settingRepo, storageRepo, cfg.StorageMgr, cfg.AuthSvc, cfg.ReloadStorage)
	systemStatusHandler := handler.NewSystemStatusRealtimeHandler(realtimeCollector)

	// Public short links and top-level non-API routes.
	registerFilePublicServe(r, fileHandler)

	v1 := r.Group("/api/v1")
	registerHealth(v1)
	registerSystemStatusStream(v1, systemStatusHandler)
	registerAuthPublic(v1, authHandler, settingRepo)
	registerSetup(v1, setupHandler)
	registerPublic(v1, fileHandler, fileRepo, settingRepo)

	authed := v1.Group("")
	authed.Use(middleware.Auth(cfg.AuthSvc))
	registerAuthAuthed(authed, authHandler)
	registerFileAuthed(authed, fileHandler)
	registerAlbumRoutes(authed, albumHandler)
	registerTokenRoutes(authed, tokenHandler)
	registerUserStatsRoutes(authed, userHandler)

	admin := authed.Group("")
	admin.Use(middleware.AdminOnly())
	registerSystemStatusAdmin(admin, systemStatusHandler)
	registerStorageAdmin(admin, storageHandler)
	registerSettingsAdmin(admin, settingsHandler)
	registerAuthAdmin(admin, oauthProviderAdminHandler)
	registerUserAdmin(admin, userHandler, fileHandler)

	registerLanding(r, cfg, userRepo, settingRepo, settingDefaults)
	registerStatic(r, cfg, settingRepo, settingDefaults)

	return r
}

// authRateLimit returns the runtime-configurable rate limit applied to
// unauthenticated auth endpoints such as login and token refresh.
func authRateLimit(settingRepo *repo.SettingRepo) gin.HandlerFunc {
	return rateLimitFromSetting(
		settingRepo,
		service.AuthRateLimitPerMinuteSettingKey,
		service.DefaultAuthRateLimitPerMinute(),
	)
}

// guestUploadRateLimit returns the runtime-configurable rate limit applied to
// the public upload endpoint. This is intentionally separate from auth routes
// because multi-file uploads require a higher burst ceiling.
func guestUploadRateLimit(settingRepo *repo.SettingRepo) gin.HandlerFunc {
	return rateLimitFromSetting(
		settingRepo,
		service.GuestUploadRateLimitPerMinuteSettingKey,
		service.DefaultGuestUploadRateLimitPerMinute(),
	)
}

func rateLimitFromSetting(settingRepo *repo.SettingRepo, key, fallback string) gin.HandlerFunc {
	fallbackLimit, err := service.ParseRequestsPerMinute(fallback)
	if err != nil {
		fallbackLimit = 1
	}

	return middleware.RateLimitFunc(time.Minute, func(c *gin.Context) int {
		raw, err := settingRepo.GetOrDefault(c.Request.Context(), key, fallback)
		if err != nil {
			return fallbackLimit
		}
		limit, err := service.ParseRequestsPerMinute(raw)
		if err != nil {
			return fallbackLimit
		}
		return limit
	})
}
