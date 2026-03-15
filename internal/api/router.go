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
	postRepository := repo.NewPostRepository(db)
	postService := service.NewPostService(postRepository)
	postHandler := NewPostHandler(postService)

	apiV1 := router.Group("/api/v1")
	apiV1.GET("/health", healthHandler.Get)
	apiV1.GET("/posts", postHandler.List)
	apiV1.GET("/posts/:id", postHandler.GetByID)
	apiV1.GET("/posts/slug/:slug", postHandler.GetBySlug)
	apiV1.POST("/posts", postHandler.Create)
	apiV1.PUT("/posts/:id", postHandler.Update)
	apiV1.PATCH("/posts/:id", postHandler.Patch)
	apiV1.DELETE("/posts/:id", postHandler.Delete)
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
