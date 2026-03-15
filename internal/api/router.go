package api

import (
	"io/fs"
	"net/http"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

func NewRouter(cfg *config.Config, templateFS fs.FS) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	registerAPIRoutes(router, cfg)
	registerPageRoutes(router, cfg, templateFS)

	return router
}

func registerAPIRoutes(router *gin.Engine, cfg *config.Config) {
	systemService := service.NewSystemService(cfg)
	healthHandler := NewHealthHandler(systemService)

	apiV1 := router.Group("/api/v1")
	apiV1.GET("/health", healthHandler.Get)
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
