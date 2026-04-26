package router

import (
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
)

// registerStatic wires the static asset serving paths and the SPA fallback:
//
//   - /static/* streams embedded landing-page assets (background images etc.)
//   - /uploads/* serves files backed by the local storage driver.
//   - NoRoute serves the embedded admin SPA and rewrites unknown paths to its
//     index.html so client-side routing works on deep links.
func registerStatic(r *gin.Engine, cfg Config, settingRepo *repo.SettingRepo, settingDefaults map[string]string) {
	if cfg.TemplateFS != nil {
		r.GET("/static/*filepath", func(c *gin.Context) {
			fp := strings.TrimPrefix(c.Param("filepath"), "/")
			if f, err := cfg.TemplateFS.Open("static/" + fp); err == nil {
				f.Close()
				c.FileFromFS("static/"+fp, http.FS(cfg.TemplateFS))
				return
			}
			c.String(http.StatusNotFound, "not found")
		})
	}

	if cfg.DataDir != "" {
		r.Static("/uploads", cfg.DataDir+"/uploads")
	}

	if cfg.AdminFS != nil {
		r.NoRoute(func(c *gin.Context) {
			fsPath := strings.TrimPrefix(c.Request.URL.Path, "/")

			if fsPath != "" {
				if f, err := cfg.AdminFS.Open(fsPath); err == nil {
					f.Close()
					c.FileFromFS(fsPath, http.FS(cfg.AdminFS))
					return
				}
			}

			data, err := fs.ReadFile(cfg.AdminFS, "index.html")
			if err != nil {
				c.String(http.StatusNotFound, "frontend not built")
				return
			}
			settings := loadResolvedSettings(c.Request.Context(), settingRepo, settingDefaults)
			title := template.HTMLEscapeString(buildAdminPageTitle(c, settings))
			faviconURL := template.HTMLEscapeString(settings[service.SiteFaviconURLSettingKey])
			html := strings.Replace(string(data), "__KITE_ADMIN_TITLE__", title, 1)
			html = strings.Replace(html, "__KITE_FAVICON_URL__", faviconURL, 1)
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		})
	}
}
