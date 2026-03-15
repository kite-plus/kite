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
	apiV1.POST("/posts", postHandler.Create)
	apiV1.PUT("/posts/:id", postHandler.Update)
	apiV1.PATCH("/posts/:id", postHandler.Patch)
	apiV1.DELETE("/posts/:id", postHandler.Delete)
	apiV1.GET("/friend-links", friendLinkHandler.List)
	apiV1.GET("/friend-links/:id", friendLinkHandler.GetByID)
	apiV1.POST("/friend-links", friendLinkHandler.Create)
	apiV1.PUT("/friend-links/:id", friendLinkHandler.Update)
	apiV1.PATCH("/friend-links/:id", friendLinkHandler.Patch)
	apiV1.DELETE("/friend-links/:id", friendLinkHandler.Delete)
	apiV1.GET("/tags", tagHandler.List)
	apiV1.GET("/tags/:id", tagHandler.GetByID)
	apiV1.POST("/tags", tagHandler.Create)
	apiV1.PUT("/tags/:id", tagHandler.Update)
	apiV1.PATCH("/tags/:id", tagHandler.Patch)
	apiV1.DELETE("/tags/:id", tagHandler.Delete)
	apiV1.GET("/categories", categoryHandler.List)
	apiV1.GET("/categories/:id", categoryHandler.GetByID)
	apiV1.POST("/categories", categoryHandler.Create)
	apiV1.PUT("/categories/:id", categoryHandler.Update)
	apiV1.PATCH("/categories/:id", categoryHandler.Patch)
	apiV1.DELETE("/categories/:id", categoryHandler.Delete)

	adminV1 := apiV1.Group("/admin")
	adminV1.GET("/posts", postHandler.List)
	adminV1.GET("/posts/:id", postHandler.GetByID)
	adminV1.GET("/posts/slug/:slug", postHandler.GetBySlug)
	adminV1.POST("/posts", postHandler.Create)
	adminV1.PUT("/posts/:id", postHandler.Update)
	adminV1.PATCH("/posts/:id", postHandler.Patch)
	adminV1.DELETE("/posts/:id", postHandler.Delete)
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
