package api

import (
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/amigoer/kite/internal/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RouterConfig 路由配置所需的依赖。
type RouterConfig struct {
	DB            *gorm.DB
	StorageMgr    *storage.Manager
	AuthSvc       *service.AuthService
	FileSvc       *service.FileService
	AdminFS       fs.FS  // 内嵌的 SPA 资产（web/admin/dist）
	TemplateFS    fs.FS  // 内嵌的 Go 模板（web/template）
	DataDir       string // 数据目录（用于本地存储文件的静态服务）
	ReloadStorage func() // 在存储配置 CRUD 后重建 Manager 状态，避免 defaultID 与 DB 不一致
}

// SetupRouter 注册所有路由并返回 gin.Engine 实例。
func SetupRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()
	realtimeCollector := NewRealtimeSystemStatusCollector()

	// 全局中间件
	r.Use(middleware.CORS())
	r.Use(realtimeCollector.Middleware())

	// 初始化 repos
	userRepo := repo.NewUserRepo(cfg.DB)
	fileRepo := repo.NewFileRepo(cfg.DB)
	albumRepo := repo.NewAlbumRepo(cfg.DB)
	tokenRepo := repo.NewAPITokenRepo(cfg.DB)
	storageRepo := repo.NewStorageConfigRepo(cfg.DB)
	settingRepo := repo.NewSettingRepo(cfg.DB)
	accessLogRepo := repo.NewFileAccessLogRepo(cfg.DB)

	// 初始化 handlers
	authHandler := NewAuthHandler(cfg.AuthSvc, userRepo)
	fileHandler := NewFileHandler(cfg.FileSvc, fileRepo, albumRepo, accessLogRepo)
	albumHandler := NewAlbumHandler(albumRepo, fileRepo)
	tokenHandler := NewTokenHandler(cfg.AuthSvc, tokenRepo)
	storageHandler := NewStorageHandler(storageRepo, fileRepo, cfg.StorageMgr, cfg.ReloadStorage)
	settingsHandler := NewSettingsHandler(settingRepo)
	userHandler := NewUserHandler(userRepo, fileRepo, accessLogRepo, cfg.AuthSvc)
	setupHandler := NewSetupHandler(userRepo, settingRepo, storageRepo, cfg.StorageMgr, cfg.AuthSvc, cfg.ReloadStorage)
	systemStatusRealtimeHandler := NewSystemStatusRealtimeHandler(realtimeCollector)

	// ========== 公开接口（无需认证）==========

	// 文件访问短链
	r.GET("/i/:hash", fileHandler.ServeImage)
	r.GET("/v/:hash", fileHandler.ServeVideo)
	r.GET("/a/:hash", fileHandler.ServeAudio)
	r.GET("/f/:hash", fileHandler.ServeDownload)
	r.GET("/t/:hash", fileHandler.ServeThumbnail)

	// API v1
	v1 := r.Group("/api/v1")
	v1.GET("/admin/system-status/ws", systemStatusRealtimeHandler.Stream)
	v1.GET("/health", func(c *gin.Context) {
		success(c, gin.H{
			"status": "ok",
			"ts":     time.Now().Unix(),
		})
	})

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

	// 公开接口（无需认证）
	pub := v1.Group("/public")
	{
		// 站点公开统计
		pub.GET("/stats", func(c *gin.Context) {
			stats, err := fileRepo.GetStats(c.Request.Context())
			if err != nil {
				serverError(c, "failed to get stats")
				return
			}
			success(c, gin.H{
				"total_files": stats.TotalFiles,
				"total_size":  stats.TotalSize,
				"images":      stats.ImageCount,
				"videos":      stats.VideoCount,
				"audios":      stats.AudioCount,
			})
		})

		// 公开图片/文件列表（探索广场）
		pub.GET("/files", func(c *gin.Context) {
			// 检查是否开启了公开广场
			val, _ := settingRepo.Get(c.Request.Context(), "allow_public_gallery")
			if val != "true" {
				fail(c, http.StatusForbidden, 40300, "public gallery is disabled")
				return
			}

			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			size, _ := strconv.Atoi(c.DefaultQuery("size", "24"))
			if page < 1 {
				page = 1
			}
			if size < 1 || size > 100 {
				size = 24
			}

			files, total, err := fileRepo.List(c.Request.Context(), repo.FileListParams{
				FileType: c.Query("file_type"),
				Page:     page,
				PageSize: size,
				OrderBy:  "created_at",
				Order:    "DESC",
			})
			if err != nil {
				serverError(c, "failed to list files")
				return
			}
			success(c, gin.H{
				"items": fileHandler.enrichFiles(files, requestBaseURL(c)),
				"total": total,
				"page":  page,
				"size":  size,
			})
		})

		// 游客上传
		uploadGroup := pub.Group("")
		uploadGroup.Use(middleware.RateLimit(10, time.Minute))
		{
			uploadGroup.POST("/upload", func(c *gin.Context) {
				val, err := settingRepo.Get(c.Request.Context(), "allow_guest_upload")
				if err != nil || val != "true" {
					fail(c, http.StatusForbidden, 40300, "guest upload is disabled")
					return
				}
				fileHandler.GuestUpload(c)
			})
		}
	}

	// ========== 需要认证的接口 ==========
	authed := v1.Group("")
	authed.Use(middleware.Auth(cfg.AuthSvc))
	{
		// 用户信息
		authed.GET("/profile", authHandler.GetProfile)
		authed.PUT("/profile", authHandler.UpdateProfile)
		authed.POST("/auth/logout", authHandler.Logout)
		authed.POST("/auth/change-password", authHandler.ChangePassword)
		authed.POST("/auth/first-login-reset", authHandler.FirstLoginReset)

		// 文件管理
		authed.POST("/upload", fileHandler.Upload)
		authed.GET("/files", fileHandler.List)
		authed.GET("/files/:id", fileHandler.Detail)
		authed.DELETE("/files/:id", fileHandler.Delete)
		authed.POST("/files/batch-delete", fileHandler.BatchDelete)
		authed.PATCH("/files/:id/move", fileHandler.MoveFile)

		// 文件夹管理（兼容旧 albums API）
		authed.GET("/albums", albumHandler.List)
		authed.POST("/albums", albumHandler.Create)
		authed.PUT("/albums/:id", albumHandler.Update)
		authed.DELETE("/albums/:id", albumHandler.Delete)
		authed.GET("/folders", albumHandler.List)
		authed.POST("/folders", albumHandler.Create)
		authed.PUT("/folders/:id", albumHandler.Update)
		authed.DELETE("/folders/:id", albumHandler.Delete)

		// API Token 管理
		authed.GET("/tokens", tokenHandler.List)
		authed.POST("/tokens", tokenHandler.Create)
		authed.DELETE("/tokens/:id", tokenHandler.Delete)

		// 使用统计
		authed.GET("/stats", userHandler.Stats)
		authed.GET("/stats/daily", userHandler.DailyStats)
		authed.GET("/stats/heatmap", userHandler.HeatmapStats)

		// ========== 管理员接口 ==========
		admin := authed.Group("")
		admin.Use(middleware.AdminOnly())
		{
			admin.POST("/admin/system-status/ws-ticket", systemStatusRealtimeHandler.IssueWSTicket)

			// 存储配置
			admin.GET("/storage", storageHandler.List)
			admin.GET("/storage/:id", storageHandler.GetOne)
			admin.POST("/storage", storageHandler.Create)
			admin.PUT("/storage/:id", storageHandler.Update)
			admin.DELETE("/storage/:id", storageHandler.Delete)
			admin.POST("/storage/:id/test", storageHandler.Test)
			admin.POST("/storage/:id/set-default", storageHandler.SetDefault)
			admin.POST("/storage/reorder", storageHandler.Reorder)

			// 系统设置
			admin.GET("/settings", settingsHandler.Get)
			admin.PUT("/settings", settingsHandler.Update)

			// 全站统计（管理员视角）
			admin.GET("/admin/stats", userHandler.AdminStats)
			admin.GET("/admin/stats/daily", userHandler.AdminDailyStats)
			admin.GET("/admin/stats/heatmap", userHandler.AdminHeatmapStats)

			// 文件管理（全站）
			admin.GET("/admin/files", fileHandler.AdminList)
			admin.DELETE("/admin/files/:id", fileHandler.AdminDelete)

			// 用户管理
			admin.GET("/admin/users", userHandler.List)
			admin.POST("/admin/users", userHandler.Create)
			admin.PUT("/admin/users/:id", userHandler.Update)
			admin.DELETE("/admin/users/:id", userHandler.Delete)
		}
	}

	// Go 模板落地页（从内嵌 FS 加载，支持单文件部署）
	if cfg.TemplateFS != nil {
		tmpl, err := template.ParseFS(cfg.TemplateFS, "layouts/*.html", "pages/*.html")
		if err == nil {
			r.SetHTMLTemplate(tmpl)
		}
	}
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"CurrentUser": getOptionalUser(c, cfg.AuthSvc, userRepo),
			"ActiveNav":   "",
		})
	})
	r.GET("/explore", func(c *gin.Context) {
		val, _ := settingRepo.Get(c.Request.Context(), "allow_public_gallery")
		c.HTML(http.StatusOK, "explore.html", gin.H{
			"GalleryEnabled": val == "true",
			"CurrentUser":    getOptionalUser(c, cfg.AuthSvc, userRepo),
			"ActiveNav":      "explore",
		})
	})
	r.GET("/upload", func(c *gin.Context) {
		// 检查是否开启了游客上传
		val, _ := settingRepo.Get(c.Request.Context(), "allow_guest_upload")
		c.HTML(http.StatusOK, "upload.html", gin.H{
			"GuestUploadEnabled": val == "true",
			"CurrentUser":        getOptionalUser(c, cfg.AuthSvc, userRepo),
			"ActiveNav":          "upload",
		})
	})

	// 前台模板静态资源（背景图等）
	if cfg.TemplateFS != nil {
		r.GET("/static/*filepath", func(c *gin.Context) {
			fp := strings.TrimPrefix(c.Param("filepath"), "/")
			if f, err := cfg.TemplateFS.Open("static/" + fp); err == nil {
				f.Close()
				c.FileFromFS("static/"+fp, http.FS(cfg.TemplateFS))
			} else {
				c.String(http.StatusNotFound, "not found")
			}
		})
	}

	// 本地存储文件直接访问（source_url）
	if cfg.DataDir != "" {
		r.Static("/uploads", cfg.DataDir+"/uploads")
	}

	// 前端 SPA 静态资源服务（用户中心 + 管理后台）
	if cfg.AdminFS != nil {
		r.NoRoute(func(c *gin.Context) {
			urlPath := c.Request.URL.Path
			fsPath := strings.TrimPrefix(urlPath, "/")

			// 尝试匹配静态文件（JS、CSS、图片等）
			if fsPath != "" {
				if f, err := cfg.AdminFS.Open(fsPath); err == nil {
					f.Close()
					c.FileFromFS(fsPath, http.FS(cfg.AdminFS))
					return
				}
			}

			// 找不到具体文件则回退到 index.html（SPA 路由）
			data, err := fs.ReadFile(cfg.AdminFS, "index.html")
			if err == nil {
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			} else {
				c.String(http.StatusNotFound, "frontend not built")
			}
		})
	}

	return r
}

// publicUser 公开页面模板注入的当前登录用户视图（仅包含展示所需字段）。
type publicUser struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	Initial     string
	Role        string
	IsAdmin     bool
}

// getOptionalUser 尝试从 access_token cookie 解析当前登录用户；无 cookie 或无效时返回 nil。
// 用于公开落地页（/ /explore /upload）在不强制登录的前提下识别登录态，避免前台模板仍显示“登录”按钮。
func getOptionalUser(c *gin.Context, authSvc *service.AuthService, userRepo *repo.UserRepo) *publicUser {
	cookie, err := c.Cookie("access_token")
	if err != nil || cookie == "" {
		return nil
	}
	claims, err := authSvc.ValidateToken(cookie)
	if err != nil {
		return nil
	}

	userView := &publicUser{
		ID:          claims.UserID,
		Username:    claims.Username,
		DisplayName: claims.Username,
		Role:        claims.Role,
		IsAdmin:     claims.Role == "admin",
	}

	if user, getErr := userRepo.GetByID(c.Request.Context(), claims.UserID); getErr == nil {
		userView.Username = user.Username
		if user.Nickname != nil {
			nickname := strings.TrimSpace(*user.Nickname)
			if nickname != "" {
				userView.DisplayName = nickname
			}
		}
		if user.AvatarURL != nil {
			userView.AvatarURL = strings.TrimSpace(*user.AvatarURL)
		}
	}

	userView.Initial = nameInitial(userView.Username)

	return userView
}

func nameInitial(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "U"
	}
	r, _ := utf8.DecodeRuneInString(trimmed)
	if r == utf8.RuneError {
		return "U"
	}
	return strings.ToUpper(string(r))
}
