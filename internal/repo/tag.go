package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrTagNotFound = errors.New("tag not found")

type TagListParams struct {
	Page     int
	PageSize int
	Keyword  string
}

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) List(params TagListParams) ([]model.Tag, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("tag repository is unavailable")
	}

	query := r.db.Model(&model.Tag{})
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		query = query.Where("name LIKE ? OR slug LIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tags: %w", err)
	}

	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	var items []model.Tag
	if err := query.Order("created_at DESC").Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list tags: %w", err)
	}

	return items, total, nil
}

func (r *TagRepository) GetByID(id uuid.UUID) (*model.Tag, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("tag repository is unavailable")
	}

	var item model.Tag
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag by id: %w", err)
	}

	return &item, nil
}

func (r *TagRepository) GetBySlug(slug string) (*model.Tag, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("tag repository is unavailable")
	}

	var item model.Tag
	if err := r.db.First(&item, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag by slug: %w", err)
	}

	return &item, nil
}

func (r *TagRepository) GetByName(name string) (*model.Tag, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("tag repository is unavailable")
	}

	var item model.Tag
	if err := r.db.First(&item, "name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("get tag by name: %w", err)
	}

	return &item, nil
}

func (r *TagRepository) Create(item *model.Tag) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("tag repository is unavailable")
	}
	if err := r.db.Create(item).Error; err != nil {
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

func (r *TagRepository) Update(item *model.Tag) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("tag repository is unavailable")
	}
	result := r.db.Model(item).Select("name", "slug").Updates(item)
	if result.Error != nil {
		return fmt.Errorf("update tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTagNotFound
	}
	return nil
}

func (r *TagRepository) Delete(id uuid.UUID) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("tag repository is unavailable")
	}
	result := r.db.Delete(&model.Tag{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrTagNotFound
	}
	return nil
}
