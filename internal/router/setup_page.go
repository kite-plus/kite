package router

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/repo"
)

// installedSetting is the row [handler.SetupHandler] writes to mark the
// instance as installed. We probe both the user count *and* this flag
// because the user table empties out when an operator wipes credentials but
// keeps the rest of the data — without the flag check the wizard would pop
// up again and be a footgun.
const installedSetting = "is_installed"

// registerSetupPage wires the server-rendered install wizard at /setup. The
// page itself is a Go template that reuses the landing layout's header and
// footer; the actual install action POSTs to the existing JSON endpoints
// under /api/v1/setup/*.
//
// The page also installs a guard on the public landing routes: when the
// system isn't installed yet, hits to /, /explore, /upload and /login are
// rewritten to /setup so a fresh instance funnels operators directly into
// the wizard rather than letting them stumble around half-empty pages.
func registerSetupPage(r *gin.Engine, cfg Config, settingRepo *repo.SettingRepo, settingDefaults map[string]string) {
	r.GET("/setup", func(c *gin.Context) {
		settings := loadResolvedSettings(c.Request.Context(), settingRepo, settingDefaults)

		// Already installed → kick the visitor to /login. Without this check
		// an attacker could keep the wizard open and clobber settings as a
		// privilege-escalation vector.
		if isInstalled(c.Request.Context(), settingRepo) {
			c.Redirect(http.StatusFound, "/login")
			return
		}

		locale := middleware.LocaleFromGin(c)
		data := landingTemplateData(c, nil, settings, "setup", i18n.T(locale, "setup_page.doc_title"))
		data["DatabaseDriver"] = cfg.CurrentDatabase.Driver
		data["DatabaseDSN"] = cfg.CurrentDatabase.DSN
		data["DatabaseSwitchEnabled"] = cfg.SaveDatabaseConfig != nil
		// Pre-fill the site URL with the canonical address the server is
		// reachable at right now — same heuristic the share page uses.
		data["SuggestedSiteURL"] = requestBaseURL(c)
		data["SuggestedStorageBasePath"] = defaultStorageBasePath(cfg.DataDir)
		c.HTML(http.StatusOK, "setup.html", data)
	})
}

// installRedirectMiddleware sends every public-page hit to /setup until the
// system is installed. It runs *before* registerLanding wires its routes, so
// the redirect wins — gin runs middleware in registration order.
//
// The redirect is scoped to a small set of well-known landing paths so we
// don't accidentally redirect API calls or static asset fetches that the
// wizard itself depends on (e.g. CSS over a CDN is fine, but /api/v1/setup/*
// must reach the JSON handlers without bouncing through HTML).
func installRedirectMiddleware(settingRepo *repo.SettingRepo) gin.HandlerFunc {
	guarded := map[string]struct{}{
		"/":        {},
		"/explore": {},
		"/upload":  {},
		"/login":   {},
	}
	return func(c *gin.Context) {
		if _, ok := guarded[c.Request.URL.Path]; !ok {
			c.Next()
			return
		}
		// /setup itself must remain reachable; the early-return below also
		// covers it via the `guarded` map filter (it's not in the set).
		if isInstalled(c.Request.Context(), settingRepo) {
			c.Next()
			return
		}
		c.Redirect(http.StatusFound, "/setup")
		c.Abort()
	}
}

// isInstalled reports whether the install flag is set in the settings table.
// Errors degrade to "not installed" — better to show the wizard one extra
// time than to lock an operator out of a fresh instance.
func isInstalled(ctx context.Context, settingRepo *repo.SettingRepo) bool {
	val, err := settingRepo.Get(ctx, installedSetting)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(val), "true")
}

// defaultStorageBasePath returns the path the wizard should pre-fill for the
// "Local" storage driver. Mirrors the seed used by main.go on first boot so
// operators see the same path they'd get if they did nothing.
func defaultStorageBasePath(dataDir string) string {
	if dataDir == "" || dataDir == "." {
		return "data/uploads"
	}
	return strings.TrimRight(dataDir, "/") + "/uploads"
}
