package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
	"github.com/kite-plus/kite/internal/service"
	"github.com/kite-plus/kite/internal/storage"
)

// SetupHandler handles the first-install wizard HTTP requests.
type SetupHandler struct {
	userRepo      *repo.UserRepo
	settingRepo   *repo.SettingRepo
	storageRepo   *repo.StorageConfigRepo
	storageMgr    *storage.Manager
	authSvc       *service.AuthService
	reloadStorage func()
}

func NewSetupHandler(
	userRepo *repo.UserRepo,
	settingRepo *repo.SettingRepo,
	storageRepo *repo.StorageConfigRepo,
	storageMgr *storage.Manager,
	authSvc *service.AuthService,
	reloadStorage func(),
) *SetupHandler {
	if reloadStorage == nil {
		reloadStorage = func() {}
	}
	return &SetupHandler{
		userRepo:      userRepo,
		settingRepo:   settingRepo,
		storageRepo:   storageRepo,
		storageMgr:    storageMgr,
		authSvc:       authSvc,
		reloadStorage: reloadStorage,
	}
}

type setupRequest struct {
	// Site configuration.
	SiteName string `json:"site_name" binding:"required"`
	SiteURL  string `json:"site_url" binding:"required"`

	// Admin account.
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=32"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=64"`

	// Storage configuration.
	StorageScheme string          `json:"storage_scheme"`
	StorageDriver string          `json:"storage_driver"` // legacy fallback
	StorageConfig json.RawMessage `json:"storage_config" binding:"required"`
}

// Setup handles the first-install request.
func (h *SetupHandler) Setup(c *gin.Context) {
	// "is_installed" is the source of truth for "have we completed the
	// install wizard?". Gating on user count would conflict with auto-seeded
	// admin accounts on fresh boots — those would falsely flag the system
	// as installed and lock the wizard out before the operator ever gets
	// to use it.
	if installed, _ := h.settingRepo.Get(c.Request.Context(), "is_installed"); strings.EqualFold(strings.TrimSpace(installed), "true") {
		BadRequest(c, M(c, i18n.KeySetupAlreadyInitialized))
		return
	}

	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeySetupInvalidData, err.Error()))
		return
	}

	ctx := c.Request.Context()

	// 1. Persist site settings.
	settings := map[string]string{
		"site_name":    req.SiteName,
		"site_url":     req.SiteURL,
		"is_installed": "true",
	}
	if err := h.settingRepo.SetBatch(ctx, settings); err != nil {
		ServerError(c, M(c, i18n.KeySetupSaveSiteSettingsFailed))
		return
	}

	// 2. Create the admin account.
	admin, err := h.authSvc.CreateAdminUser(ctx, req.AdminUsername, req.AdminEmail, req.AdminPassword, false)
	if err != nil {
		ServerError(c, M(c, i18n.KeySetupCreateAdminFailed, err.Error()))
		return
	}

	// 3. Create the default storage configuration.
	schemeKey := req.StorageScheme
	if schemeKey == "" {
		schemeKey = req.StorageDriver
	}
	driver, provider, normalizedConfig, scfg, err := storage.ResolveSchemeConfig(schemeKey, req.StorageConfig)
	if err != nil {
		BadRequest(c, M(c, i18n.KeySetupInvalidStorage, schemeKey, err.Error()))
		return
	}

	// Validate the storage configuration.
	if _, err := storage.NewDriver(scfg); err != nil {
		BadRequest(c, M(c, i18n.KeySetupStorageValidationFailed, err.Error()))
		return
	}

	storageCfg := &model.StorageConfig{
		ID:        uuid.New().String(),
		Name:      "Default Storage",
		Driver:    driver,
		Provider:  provider,
		Config:    string(normalizedConfig),
		Priority:  100,
		IsDefault: true,
		IsActive:  true,
	}

	if err := h.storageRepo.Create(ctx, storageCfg); err != nil {
		ServerError(c, M(c, i18n.KeySetupCreateStorageFailed))
		return
	}

	// Reload the storage manager so it picks up the default and all active configs.
	h.reloadStorage()

	Success(c, gin.H{
		"message":  "setup completed successfully",
		"admin_id": admin.ID,
	})
}

// CheckSetup reports whether the install wizard has been completed. We read
// the persisted setting flag rather than the user table count: on a fresh
// boot with seeding disabled the table is empty *and* the flag is unset (→
// install not yet run); on an upgraded existing install the table is full
// *and* the flag is set by the boot-time migration (→ install complete).
// Coupling the answer to user count instead would mis-classify both halves.
func (h *SetupHandler) CheckSetup(c *gin.Context) {
	val, _ := h.settingRepo.Get(c.Request.Context(), "is_installed")
	Success(c, gin.H{
		"is_installed": strings.EqualFold(strings.TrimSpace(val), "true"),
	})
}
