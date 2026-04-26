package router

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kite-plus/kite/internal/handler"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/version"
	"gorm.io/gorm"
)

// registerHealth wires the three cloud-native probes:
//
//   - GET /health  — liveness. Proves the HTTP stack is alive. No dependency
//     checks, no DB round-trip; safe to poll at high frequency from LBs.
//   - GET /ready   — readiness. Pings the database with a short timeout;
//     returns 503 when a dependency is unreachable so orchestrators can pull
//     the instance out of rotation during a hiccup.
//   - GET /version — build identity (version/commit/date/go). Separated from
//     /health so operators can inspect the running build without conflating
//     it with probe semantics.
func registerHealth(v1 *gin.RouterGroup, db *gorm.DB) {
	v1.GET("/health", func(c *gin.Context) {
		handler.Success(c, gin.H{
			"status": "ok",
			"ts":     time.Now().Unix(),
		})
	})

	v1.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		checks := gin.H{"db": "ok"}
		ready := true
		if sqlDB, err := db.DB(); err != nil {
			checks["db"] = "error: " + err.Error()
			ready = false
		} else if err := sqlDB.PingContext(ctx); err != nil {
			checks["db"] = "error: " + err.Error()
			ready = false
		}

		body := gin.H{
			"ts":     time.Now().Unix(),
			"checks": checks,
		}
		if !ready {
			body["status"] = "not_ready"
			c.JSON(http.StatusServiceUnavailable, handler.Response{
				Code:    50301,
				Message: "dependency unavailable",
				Data:    body,
			})
			return
		}
		body["status"] = "ready"
		handler.Success(c, body)
	})

	v1.GET("/version", func(c *gin.Context) {
		handler.Success(c, version.Get())
	})
}

// registerSetup wires the first-run installation wizard endpoints. All
// endpoints are intentionally unauthenticated because the instance has no
// user yet on first boot — they instead self-gate by checking the user
// table's row count and refuse to do anything once a user exists.
//
// The endpoints split into two groups:
//
//   - GET /setup/status  + POST /setup
//     The "happy path" — the wizard collects site / admin / storage and
//     POSTs the lot in one shot once the database driver is set in stone.
//
//   - POST /setup/test-database + POST /setup/database
//     The pre-install database stage — operators can validate a candidate
//     DSN without committing, then save it (which writes a config file and
//     prompts the operator to restart so the new driver opens at boot).
func registerSetup(v1 *gin.RouterGroup, h *handler.SetupHandler, dbH *handler.SetupDatabaseHandler) {
	v1.GET("/setup/status", h.CheckSetup)
	v1.POST("/setup", h.Setup)
	if dbH != nil {
		v1.POST("/setup/test-database", dbH.TestDatabase)
		v1.POST("/setup/database", dbH.SaveDatabase)
	}
}

// registerPublic wires the opt-in public endpoints that expose the instance
// to anonymous visitors: aggregate stats, the public gallery and the guest
// upload form. Each feature is gated by a setting flag that administrators
// can toggle from the admin panel.
func registerPublic(
	v1 *gin.RouterGroup,
	fileHandler *handler.FileHandler,
	fileRepo *repo.FileRepo,
	settingRepo *repo.SettingRepo,
) {
	pub := v1.Group("/public")

	pub.GET("/stats", func(c *gin.Context) {
		stats, err := fileRepo.GetStats(c.Request.Context())
		if err != nil {
			handler.ServerError(c, handler.M(c, i18n.KeySettingsStatsFailed))
			return
		}
		handler.Success(c, gin.H{
			"total_files": stats.TotalFiles,
			"total_size":  stats.TotalSize,
			"images":      stats.ImageCount,
			"videos":      stats.VideoCount,
			"audios":      stats.AudioCount,
		})
	})

	pub.GET("/files", func(c *gin.Context) {
		val, _ := settingRepo.Get(c.Request.Context(), "allow_public_gallery")
		if val != "true" {
			handler.Fail(c, http.StatusForbidden, 40300, handler.M(c, i18n.KeyPublicGalleryDisabled))
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
			handler.ServerError(c, handler.M(c, i18n.KeyFileListFailed))
			return
		}
		handler.Success(c, gin.H{
			"items": fileHandler.EnrichFiles(c.Request.Context(), files, handler.RequestBaseURL(c)),
			"total": total,
			"page":  page,
			"size":  size,
		})
	})

	uploadGroup := pub.Group("")
	uploadGroup.Use(guestUploadRateLimit(settingRepo))
	uploadGroup.POST("/upload", func(c *gin.Context) {
		val, err := settingRepo.Get(c.Request.Context(), "allow_guest_upload")
		if err != nil || val != "true" {
			handler.Fail(c, http.StatusForbidden, 40300, handler.M(c, i18n.KeyPublicGuestUploadDisabled))
			return
		}
		fileHandler.GuestUpload(c)
	})
}
