package repo

import (
	"github.com/amigoer/kite-blog/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingsRepository 系统设置仓库
type SettingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get 获取指定 key 的值，不存在返回空字符串
func (r *SettingsRepository) Get(key string) string {
	var s model.Setting
	if err := r.db.Where("key = ?", key).First(&s).Error; err != nil {
		return ""
	}
	return s.Value
}

// Set 设置 key-value（存在则更新，不存在则插入）
func (r *SettingsRepository) Set(key, value string) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&model.Setting{Key: key, Value: value}).Error
}

// GetAll 获取全部设置
func (r *SettingsRepository) GetAll() map[string]string {
	var settings []model.Setting
	r.db.Find(&settings)
	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result
}

// SetBatch 批量设置
func (r *SettingsRepository) SetBatch(kvs map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for k, v := range kvs {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value"}),
			}).Create(&model.Setting{Key: k, Value: v}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
