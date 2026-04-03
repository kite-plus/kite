package repo

import (
	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SlugHistoryRepository slug 变更历史仓库
type SlugHistoryRepository struct {
	db *gorm.DB
}

func NewSlugHistoryRepository(db *gorm.DB) *SlugHistoryRepository {
	return &SlugHistoryRepository{db: db}
}

// Record 记录旧 slug
func (r *SlugHistoryRepository) Record(postID uuid.UUID, oldSlug string) error {
	history := &model.SlugHistory{
		PostID:  postID,
		OldSlug: oldSlug,
	}
	// 如果已存在则忽略
	return r.db.Where("old_slug = ?", oldSlug).FirstOrCreate(history).Error
}

// FindBySlug 根据旧 slug 查找记录
func (r *SlugHistoryRepository) FindBySlug(slug string) (*model.SlugHistory, error) {
	var history model.SlugHistory
	if err := r.db.Where("old_slug = ?", slug).First(&history).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

// DeleteByPostID 删除指定文章的所有 slug 历史
func (r *SlugHistoryRepository) DeleteByPostID(postID uuid.UUID) error {
	return r.db.Where("post_id = ?", postID).Delete(&model.SlugHistory{}).Error
}
