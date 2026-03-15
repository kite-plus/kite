package api

import (
	"io/fs"
	"net/http"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(cfg *config.Config, templateFS fs.FS, db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	registerAPIRoutes(router, cfg, db)
	registerPageRoutes(router, cfg, templateFS)

	return router
}

func registerAPIRoutes(router *gin.Engine, cfg *config.Config, db *gorm.DB) {
	systemService := service.NewSystemService(cfg)
	healthHandler := NewHealthHandler(systemService)
	tagRepository := repo.NewTagRepository(db)
	categoryRepository := repo.NewCategoryRepository(db)
	postRepository := repo.NewPostRepository(db)
	postService := service.NewPostService(postRepository, tagRepository, categoryRepository)
	postHandler := NewPostHandler(postService)
	adminSessionRepository := repo.NewAdminSessionRepository(db)
	adminAuthService := service.NewAdminAuthService(cfg, adminSessionRepository)
	adminAuthHandler := NewAdminAuthHandler(adminAuthService)
	adminAuthMiddleware := NewAdminAuthMiddleware(adminAuthService)
	friendLinkRepository := repo.NewFriendLinkRepository(db)
	friendLinkService := service.NewFriendLinkService(friendLinkRepository)
	friendLinkHandler := NewFriendLinkHandler(friendLinkService)
	tagService := service.NewTagService(tagRepository)
	tagHandler := NewTagHandler(tagService)
	categoryService := service.NewCategoryService(categoryRepository)
	categoryHandler := NewCategoryHandler(categoryService)

	apiV1 := router.Group("/api/v1")
	apiV1.GET("/health", healthHandler.Get)
	apiV1.GET("/posts", postHandler.ListPublic)
	apiV1.GET("/posts/:id", postHandler.GetPublicByID)
	apiV1.GET("/posts/slug/:slug", postHandler.GetPublicBySlug)
	apiV1.GET("/friend-links", friendLinkHandler.ListPublic)
	apiV1.GET("/friend-links/:id", friendLinkHandler.GetPublicByID)
	apiV1.GET("/tags", tagHandler.List)
	apiV1.GET("/tags/:id", tagHandler.GetByID)
	apiV1.GET("/categories", categoryHandler.List)
	apiV1.GET("/categories/:id", categoryHandler.GetByID)

	adminV1 := apiV1.Group("/admin")
	adminV1.POST("/auth/login", adminAuthHandler.Login)
	adminV1.GET("/auth/me", adminAuthHandler.Me)

	protectedAdminV1 := adminV1.Group("/")
	protectedAdminV1.Use(adminAuthMiddleware.Require())
	protectedAdminV1.POST("/auth/logout", adminAuthHandler.Logout)
	protectedAdminV1.GET("/posts", postHandler.List)
	protectedAdminV1.GET("/posts/:id", postHandler.GetByID)
	protectedAdminV1.GET("/posts/slug/:slug", postHandler.GetBySlug)
	protectedAdminV1.POST("/posts", postHandler.Create)
	protectedAdminV1.PUT("/posts/:id", postHandler.Update)
	protectedAdminV1.PATCH("/posts/:id", postHandler.Patch)
	protectedAdminV1.DELETE("/posts/:id", postHandler.Delete)
	protectedAdminV1.GET("/friend-links", friendLinkHandler.List)
	protectedAdminV1.GET("/friend-links/:id", friendLinkHandler.GetByID)
	protectedAdminV1.POST("/friend-links", friendLinkHandler.Create)
	protectedAdminV1.PUT("/friend-links/:id", friendLinkHandler.Update)
	protectedAdminV1.PATCH("/friend-links/:id", friendLinkHandler.Patch)
	protectedAdminV1.DELETE("/friend-links/:id", friendLinkHandler.Delete)
	protectedAdminV1.GET("/tags", tagHandler.List)
	protectedAdminV1.GET("/tags/:id", tagHandler.GetByID)
	protectedAdminV1.POST("/tags", tagHandler.Create)
	protectedAdminV1.PUT("/tags/:id", tagHandler.Update)
	protectedAdminV1.PATCH("/tags/:id", tagHandler.Patch)
	protectedAdminV1.DELETE("/tags/:id", tagHandler.Delete)
	protectedAdminV1.GET("/categories", categoryHandler.List)
	protectedAdminV1.GET("/categories/:id", categoryHandler.GetByID)
	protectedAdminV1.POST("/categories", categoryHandler.Create)
	protectedAdminV1.PUT("/categories/:id", categoryHandler.Update)
	protectedAdminV1.PATCH("/categories/:id", categoryHandler.Patch)
	protectedAdminV1.DELETE("/categories/:id", categoryHandler.Delete)
}

func registerPageRoutes(router *gin.Engine, cfg *config.Config, templateFS fs.FS) {
	if cfg == nil || cfg.RenderMode != config.RenderModeClassic {
		return
	}

	if templateFS == nil {
		return
	}

	tmpl, err := loadTemplateSet(templateFS)
	if err != nil {
		return
	}

	router.SetHTMLTemplate(tmpl)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"Title":      "Kite",
			"RenderMode": cfg.RenderMode,
		})
	})
}
