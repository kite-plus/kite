package repo

import (
	"context"
	"fmt"

	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/storage"
	"gorm.io/gorm"
)

// StorageConfigRepo 存储配置数据访问层。
type StorageConfigRepo struct {
	db *gorm.DB
}

func NewStorageConfigRepo(db *gorm.DB) *StorageConfigRepo {
	return &StorageConfigRepo{db: db}
}

// Create 创建存储配置。
func (r *StorageConfigRepo) Create(ctx context.Context, cfg *model.StorageConfig) error {
	if err := r.db.WithContext(ctx).Create(cfg).Error; err != nil {
		return fmt.Errorf("create storage config: %w", err)
	}
	return nil
}

// GetByID 通过 ID 查询存储配置。
func (r *StorageConfigRepo) GetByID(ctx context.Context, id string) (*model.StorageConfig, error) {
	var cfg model.StorageConfig
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("get storage config: %w", err)
	}
	return &cfg, nil
}

// GetDefault 获取默认存储配置。
func (r *StorageConfigRepo) GetDefault(ctx context.Context) (*model.StorageConfig, error) {
	var cfg model.StorageConfig
	if err := r.db.WithContext(ctx).
		Where("is_default = ? AND is_active = ?", true, true).
		First(&cfg).Error; err != nil {
		return nil, fmt.Errorf("get default storage config: %w", err)
	}
	return &cfg, nil
}

// Update 更新存储配置。
func (r *StorageConfigRepo) Update(ctx context.Context, cfg *model.StorageConfig) error {
	if err := r.db.WithContext(ctx).Save(cfg).Error; err != nil {
		return fmt.Errorf("update storage config: %w", err)
	}
	return nil
}

// SetDefault 将指定配置设为默认，同时取消其他配置的默认状态。
func (r *StorageConfigRepo) SetDefault(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取消当前所有默认
		if err := tx.Model(&model.StorageConfig{}).
			Where("is_default = ?", true).
			Update("is_default", false).Error; err != nil {
			return fmt.Errorf("clear default storage config: %w", err)
		}
		// 设置新默认
		if err := tx.Model(&model.StorageConfig{}).
			Where("id = ?", id).
			Update("is_default", true).Error; err != nil {
			return fmt.Errorf("set default storage config: %w", err)
		}
		return nil
	})
}

// Delete 删除存储配置。
func (r *StorageConfigRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.StorageConfig{}).Error; err != nil {
		return fmt.Errorf("delete storage config: %w", err)
	}
	return nil
}

// List 查询所有存储配置，按 priority 升序、created_at 次之排序。
func (r *StorageConfigRepo) List(ctx context.Context) ([]model.StorageConfig, error) {
	var configs []model.StorageConfig
	if err := r.db.WithContext(ctx).
		Order("priority ASC, created_at ASC").
		Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("list storage configs: %w", err)
	}
	return configs, nil
}

// ListActive 查询所有启用的存储配置，按 priority 升序、created_at 次之排序。
func (r *StorageConfigRepo) ListActive(ctx context.Context) ([]model.StorageConfig, error) {
	var configs []model.StorageConfig
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("priority ASC, created_at ASC").
		Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("list active storage configs: %w", err)
	}
	return configs, nil
}

// Reorder 根据给定 ID 顺序重写 priority：第一个为 100，依次 +100。
// 使用较大步长，避免后续单独调整 priority 时需要再次全表 reindex。
func (r *StorageConfigRepo) Reorder(ctx context.Context, orderedIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			priority := (i + 1) * 100
			if err := tx.Model(&model.StorageConfig{}).
				Where("id = ?", id).
				Update("priority", priority).Error; err != nil {
				return fmt.Errorf("reorder storage %q: %w", id, err)
			}
		}
		return nil
	})
}

// BuildRawConfigs 将当前所有存储配置转换为 storage.RawConfig 列表，供 Manager.Reload 使用。
// 包括未 active 的配置，Reload 会内部过滤；这样未来如果要切换启停状态也只需再次 Reload。
func (r *StorageConfigRepo) BuildRawConfigs(ctx context.Context) ([]storage.RawConfig, error) {
	configs, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]storage.RawConfig, 0, len(configs))
	for _, c := range configs {
		out = append(out, storage.RawConfig{
			ID:                 c.ID,
			Name:               c.Name,
			Driver:             c.Driver,
			ConfigJSON:         c.Config,
			Priority:           c.Priority,
			CapacityLimitBytes: c.CapacityLimitBytes,
			IsDefault:          c.IsDefault,
			IsActive:           c.IsActive,
		})
	}
	return out, nil
}
