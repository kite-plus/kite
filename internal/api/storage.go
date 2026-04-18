package api

import (
	"bytes"
	"encoding/json"

	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StorageHandler 存储配置管理的 HTTP 处理器。
type StorageHandler struct {
	storageRepo   *repo.StorageConfigRepo
	fileRepo      *repo.FileRepo
	storageMgr    *storage.Manager
	reloadStorage func() // 任何变更后调用以刷新 Manager 状态
}

func NewStorageHandler(
	storageRepo *repo.StorageConfigRepo,
	fileRepo *repo.FileRepo,
	storageMgr *storage.Manager,
	reloadStorage func(),
) *StorageHandler {
	if reloadStorage == nil {
		reloadStorage = func() {}
	}
	return &StorageHandler{
		storageRepo:   storageRepo,
		fileRepo:      fileRepo,
		storageMgr:    storageMgr,
		reloadStorage: reloadStorage,
	}
}

type createStorageRequest struct {
	Name               string          `json:"name" binding:"required"`
	Driver             string          `json:"driver" binding:"required,oneof=local s3 oss cos ftp"`
	Config             json.RawMessage `json:"config" binding:"required"`
	CapacityLimitBytes int64           `json:"capacity_limit_bytes"`
	Priority           int             `json:"priority"`
	IsDefault          bool            `json:"is_default"`
	IsActive           *bool           `json:"is_active"` // 指针以区分未提供与显式 false
}

// Create 添加存储配置。
func (h *StorageHandler) Create(c *gin.Context) {
	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid storage config: "+err.Error())
		return
	}

	scfg, err := storage.ParseConfig(req.Driver, req.Config)
	if err != nil {
		badRequest(c, "invalid "+req.Driver+" config: "+err.Error())
		return
	}

	// 验证能否创建驱动
	if _, err := storage.NewDriver(scfg); err != nil {
		badRequest(c, "storage config validation failed: "+err.Error())
		return
	}

	if req.CapacityLimitBytes < 0 {
		badRequest(c, "capacity_limit_bytes must not be negative")
		return
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 100
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	cfg := &model.StorageConfig{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Driver:             req.Driver,
		Config:             string(req.Config),
		CapacityLimitBytes: req.CapacityLimitBytes,
		Priority:           priority,
		IsDefault:          req.IsDefault,
		IsActive:           isActive,
	}

	if err := h.storageRepo.Create(c.Request.Context(), cfg); err != nil {
		serverError(c, "failed to create storage config")
		return
	}

	// 如果前端要求将此配置设为默认，先清除其他默认再重载，避免 Manager 读到两个默认。
	if req.IsDefault {
		if err := h.storageRepo.SetDefault(c.Request.Context(), cfg.ID); err != nil {
			serverError(c, "failed to set default storage: "+err.Error())
			return
		}
	}

	h.reloadStorage()

	created(c, gin.H{
		"id":     cfg.ID,
		"name":   cfg.Name,
		"driver": cfg.Driver,
	})
}

// List 获取所有存储配置。
func (h *StorageHandler) List(c *gin.Context) {
	configs, err := h.storageRepo.List(c.Request.Context())
	if err != nil {
		serverError(c, "failed to list storage configs")
		return
	}

	// 不返回敏感的 config 字段内容；provider 由 driver+endpoint 推断，仅用于前端渲染品牌 logo。
	type item struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		Driver             string `json:"driver"`
		Provider           string `json:"provider"`
		CapacityLimitBytes int64  `json:"capacity_limit_bytes"`
		UsedBytes          int64  `json:"used_bytes"`
		Priority           int    `json:"priority"`
		IsDefault          bool   `json:"is_default"`
		IsActive           bool   `json:"is_active"`
	}
	items := make([]item, len(configs))
	for i, cfg := range configs {
		used, _ := h.fileRepo.SumSizeByStorageConfig(c.Request.Context(), cfg.ID)
		items[i] = item{
			ID:                 cfg.ID,
			Name:               cfg.Name,
			Driver:             cfg.Driver,
			Provider:           storage.DetectProvider(cfg.Driver, cfg.Config),
			CapacityLimitBytes: cfg.CapacityLimitBytes,
			UsedBytes:          used,
			Priority:           cfg.Priority,
			IsDefault:          cfg.IsDefault,
			IsActive:           cfg.IsActive,
		}
	}

	success(c, items)
}

// GetOne 获取单个存储配置的完整详情（含 config JSON），用于编辑对话框回显。
func (h *StorageHandler) GetOne(c *gin.Context) {
	id := c.Param("id")

	cfg, err := h.storageRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		notFound(c, "storage config not found")
		return
	}

	used, _ := h.fileRepo.SumSizeByStorageConfig(c.Request.Context(), id)

	var configJSON json.RawMessage
	if cfg.Config != "" {
		configJSON = json.RawMessage(cfg.Config)
	}

	success(c, gin.H{
		"id":                   cfg.ID,
		"name":                 cfg.Name,
		"driver":               cfg.Driver,
		"config":               configJSON,
		"capacity_limit_bytes": cfg.CapacityLimitBytes,
		"used_bytes":           used,
		"priority":             cfg.Priority,
		"is_default":           cfg.IsDefault,
		"is_active":            cfg.IsActive,
	})
}

// Update 更新存储配置。
func (h *StorageHandler) Update(c *gin.Context) {
	id := c.Param("id")

	existing, err := h.storageRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		notFound(c, "storage config not found")
		return
	}

	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid storage config: "+err.Error())
		return
	}

	if req.CapacityLimitBytes < 0 {
		badRequest(c, "capacity_limit_bytes must not be negative")
		return
	}

	scfg, err := storage.ParseConfig(req.Driver, req.Config)
	if err != nil {
		badRequest(c, "invalid "+req.Driver+" config: "+err.Error())
		return
	}
	if _, err := storage.NewDriver(scfg); err != nil {
		badRequest(c, "storage config validation failed: "+err.Error())
		return
	}

	existing.Name = req.Name
	existing.Driver = req.Driver
	existing.Config = string(req.Config)
	existing.CapacityLimitBytes = req.CapacityLimitBytes
	if req.Priority > 0 {
		existing.Priority = req.Priority
	}
	// is_default 在这里不直接落库，避免绕过 SetDefault 的互斥语义；由下方 SetDefault 统一处理。
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.storageRepo.Update(c.Request.Context(), existing); err != nil {
		serverError(c, "failed to update storage config")
		return
	}

	if req.IsDefault && !existing.IsDefault {
		if err := h.storageRepo.SetDefault(c.Request.Context(), id); err != nil {
			serverError(c, "failed to set default storage: "+err.Error())
			return
		}
	}

	h.reloadStorage()

	success(c, gin.H{"id": id, "name": req.Name, "driver": req.Driver})
}

// Delete 删除存储配置。
func (h *StorageHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.storageRepo.Delete(c.Request.Context(), id); err != nil {
		serverError(c, "failed to delete storage config")
		return
	}

	h.reloadStorage()
	success(c, nil)
}

// Test 测试存储配置是否可用。
func (h *StorageHandler) Test(c *gin.Context) {
	id := c.Param("id")

	cfg, err := h.storageRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		notFound(c, "storage config not found")
		return
	}

	scfg, err := storage.ParseConfig(cfg.Driver, json.RawMessage(cfg.Config))
	if err != nil {
		serverError(c, "failed to parse storage config")
		return
	}

	driver, err := storage.NewDriver(scfg)
	if err != nil {
		success(c, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// 尝试上传一个测试文件
	testKey := ".kite-test-connection"
	testPayload := []byte("kite storage test")
	err = driver.Put(c.Request.Context(), testKey, bytes.NewReader(testPayload), int64(len(testPayload)), "text/plain")
	if err != nil {
		success(c, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// 清理测试文件
	_ = driver.Delete(c.Request.Context(), testKey)

	success(c, gin.H{"ok": true})
}

// SetDefault 将指定存储设为默认。
func (h *StorageHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	if _, err := h.storageRepo.GetByID(c.Request.Context(), id); err != nil {
		notFound(c, "storage config not found")
		return
	}
	if err := h.storageRepo.SetDefault(c.Request.Context(), id); err != nil {
		serverError(c, "failed to set default storage: "+err.Error())
		return
	}
	h.reloadStorage()
	success(c, gin.H{"id": id})
}

type reorderRequest struct {
	OrderedIDs []string `json:"ordered_ids" binding:"required"`
}

// Reorder 按前端给出的 ID 顺序批量重写 priority（拖拽排序）。
func (h *StorageHandler) Reorder(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid reorder payload: "+err.Error())
		return
	}
	if len(req.OrderedIDs) == 0 {
		badRequest(c, "ordered_ids must not be empty")
		return
	}
	if err := h.storageRepo.Reorder(c.Request.Context(), req.OrderedIDs); err != nil {
		serverError(c, "failed to reorder storages: "+err.Error())
		return
	}
	h.reloadStorage()
	success(c, nil)
}
