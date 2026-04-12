package api

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/amigoer/kite/internal/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RouterConfig 路由配置所需的依赖。
type RouterConfig struct {
	DB         *gorm.DB
	StorageMgr *storage.Manager
	AuthSvc    *service.AuthService
	FileSvc    *service.FileService
	AdminFS    fs.FS // 内嵌的前端资产，nil 时不提供前端服务
}

// SetupRouter 注册所有路由并返回 gin.Engine 实例。
func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()

	// 全局中间件
	r.Use(middleware.CORS())

	// 初始化 repos
	userRepo := repo.NewUserRepo(cfg.DB)
	fileRepo := repo.NewFileRepo(cfg.DB)
	albumRepo := repo.NewAlbumRepo(cfg.DB)
	tokenRepo := repo.NewAPITokenRepo(cfg.DB)
	storageRepo := repo.NewStorageConfigRepo(cfg.DB)
	settingRepo := repo.NewSettingRepo(cfg.DB)

	// 初始化 handlers
	authHandler := NewAuthHandler(cfg.AuthSvc)
	fileHandler := NewFileHandler(cfg.FileSvc, fileRepo)
	albumHandler := NewAlbumHandler(albumRepo, fileRepo)
	tokenHandler := NewTokenHandler(cfg.AuthSvc, tokenRepo)
	storageHandler := NewStorageHandler(storageRepo, cfg.StorageMgr)
	settingsHandler := NewSettingsHandler(settingRepo)
	userHandler := NewUserHandler(userRepo, fileRepo, cfg.AuthSvc)
	setupHandler := NewSetupHandler(userRepo, settingRepo, storageRepo, cfg.StorageMgr, cfg.AuthSvc)

	// ========== 公开接口（无需认证）==========

	// 文件访问短链
	r.GET("/i/:hash", fileHandler.ServeImage)
	r.GET("/v/:hash", fileHandler.ServeVideo)
	r.GET("/a/:hash", fileHandler.ServeAudio)
	r.GET("/f/:hash", fileHandler.ServeDownload)
	r.GET("/t/:hash", fileHandler.ServeThumbnail)

	// API v1
	v1 := r.Group("/api/v1")

	// 认证（带速率限制）
	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.RateLimit(20, time.Minute))
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// 安装向导
	v1.GET("/setup/status", setupHandler.CheckSetup)
	v1.POST("/setup", setupHandler.Setup)

	// ========== 需要认证的接口 ==========
	authed := v1.Group("")
	authed.Use(middleware.Auth(cfg.AuthSvc))
	{
		// 用户信息
		authed.GET("/profile", authHandler.GetProfile)
		authed.POST("/auth/logout", authHandler.Logout)

		// 文件管理
		authed.POST("/upload", fileHandler.Upload)
		authed.GET("/files", fileHandler.List)
		authed.GET("/files/:id", fileHandler.Detail)
		authed.DELETE("/files/:id", fileHandler.Delete)
		authed.POST("/files/batch-delete", fileHandler.BatchDelete)

		// 相册管理
		authed.GET("/albums", albumHandler.List)
		authed.POST("/albums", albumHandler.Create)
		authed.PUT("/albums/:id", albumHandler.Update)
		authed.DELETE("/albums/:id", albumHandler.Delete)

		// API Token 管理
		authed.GET("/tokens", tokenHandler.List)
		authed.POST("/tokens", tokenHandler.Create)
		authed.DELETE("/tokens/:id", tokenHandler.Delete)

		// 使用统计
		authed.GET("/stats", userHandler.Stats)

		// ========== 管理员接口 ==========
		admin := authed.Group("")
		admin.Use(middleware.AdminOnly())
		{
			// 存储配置
			admin.GET("/storage", storageHandler.List)
			admin.POST("/storage", storageHandler.Create)
			admin.PUT("/storage/:id", storageHandler.Update)
			admin.DELETE("/storage/:id", storageHandler.Delete)
			admin.POST("/storage/:id/test", storageHandler.Test)

			// 系统设置
			admin.GET("/settings", settingsHandler.Get)
			admin.PUT("/settings", settingsHandler.Update)

			// 用户管理
			admin.GET("/admin/users", userHandler.List)
			admin.POST("/admin/users", userHandler.Create)
			admin.PUT("/admin/users/:id", userHandler.Update)
			admin.DELETE("/admin/users/:id", userHandler.Delete)
		}
	}

	// 前端静态资源服务（生产模式，前端内嵌到二进制中）
	if cfg.AdminFS != nil {
		// SPA: 所有未匹配的路由返回 index.html
		r.NoRoute(func(c *gin.Context) {
			// 先尝试提供静态文件
			path := c.Request.URL.Path
			f, err := cfg.AdminFS.Open(path[1:]) // 去掉开头的 /
			if err == nil {
				f.Close()
				c.FileFromFS(path, http.FS(cfg.AdminFS))
				return
			}
			// 回退到 index.html（SPA 路由）
			c.FileFromFS("index.html", http.FS(cfg.AdminFS))
		})
	}

	return r
}
