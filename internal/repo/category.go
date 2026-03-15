package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrCategoryNotFound = errors.New("category not found")

type CategoryListParams struct {
	Page     int
	PageSize int
	Keyword  string
}

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) List(params CategoryListParams) ([]model.Category, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("category repository is unavailable")
	}

	query := r.db.Model(&model.Category{})
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		query = query.Where("name LIKE ? OR slug LIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count categories: %w", err)
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

	var items []model.Category
	if err := query.Order("created_at DESC").Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list categories: %w", err)
	}

	return items, total, nil
}

func (r *CategoryRepository) GetByID(id uuid.UUID) (*model.Category, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("category repository is unavailable")
	}

	var item model.Category
	if err := r.db.First(&item, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category by id: %w", err)
	}

	return &item, nil
}

func (r *CategoryRepository) GetBySlug(slug string) (*model.Category, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("category repository is unavailable")
	}

	var item model.Category
	if err := r.db.First(&item, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category by slug: %w", err)
	}

	return &item, nil
}

func (r *CategoryRepository) Create(item *model.Category) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("category repository is unavailable")
	}
	if err := r.db.Create(item).Error; err != nil {
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Update(item *model.Category) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("category repository is unavailable")
	}
	result := r.db.Model(item).Select("name", "slug").Updates(item)
	if result.Error != nil {
		return fmt.Errorf("update category: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("category repository is unavailable")
	}
	result := r.db.Delete(&model.Category{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete category: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}
