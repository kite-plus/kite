package repo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrPageNotFound = errors.New("page not found")

// PageListParams 页面列表查询参数
type PageListParams struct {
	Page       int
	PageSize   int
	Status     string
	Keyword    string
	PublicOnly bool
}

// PageRepository 独立页面数据仓库
type PageRepository struct {
	db *gorm.DB
}

func NewPageRepository(db *gorm.DB) *PageRepository {
	return &PageRepository{db: db}
}

// List 查询页面列表
func (r *PageRepository) List(params PageListParams) ([]model.Page, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("page repository is unavailable")
	}

	query := r.db.Model(&model.Page{})

	if params.PublicOnly {
		query = query.Where("status = ? AND (published_at IS NULL OR published_at <= ?)",
			model.PageStatusPublished, time.Now().UTC())
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		query = query.Where("title LIKE ? OR slug LIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count pages: %w", err)
	}

	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	var pages []model.Page
	if err := query.Order("sort_order ASC, created_at DESC").
		Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).
		Find(&pages).Error; err != nil {
		return nil, 0, fmt.Errorf("list pages: %w", err)
	}

	return pages, total, nil
}

// GetByID 根据 ID 获取页面
func (r *PageRepository) GetByID(id uuid.UUID) (*model.Page, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("page repository is unavailable")
	}

	var page model.Page
	if err := r.db.First(&page, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPageNotFound
		}
		return nil, fmt.Errorf("get page by id: %w", err)
	}

	return &page, nil
}

// GetBySlug 根据 slug 获取页面
func (r *PageRepository) GetBySlug(slug string) (*model.Page, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("page repository is unavailable")
	}

	var page model.Page
	if err := r.db.First(&page, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPageNotFound
		}
		return nil, fmt.Errorf("get page by slug: %w", err)
	}

	return &page, nil
}

// GetPublicBySlug 根据 slug 获取已发布页面
func (r *PageRepository) GetPublicBySlug(slug string) (*model.Page, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("page repository is unavailable")
	}

	var page model.Page
	if err := r.db.Where("status = ? AND (published_at IS NULL OR published_at <= ?)",
		model.PageStatusPublished, time.Now().UTC()).
		First(&page, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPageNotFound
		}
		return nil, fmt.Errorf("get public page by slug: %w", err)
	}

	return &page, nil
}

// Create 创建页面
func (r *PageRepository) Create(page *model.Page) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("page repository is unavailable")
	}

	if err := r.db.Create(page).Error; err != nil {
		return fmt.Errorf("create page: %w", err)
	}
	return nil
}

// Update 更新页面
func (r *PageRepository) Update(page *model.Page) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("page repository is unavailable")
	}

	result := r.db.Model(page).
		Select("title", "slug", "content_markdown", "content_html", "status", "sort_order", "show_in_nav", "published_at", "template", "config").
		Updates(page)
	if result.Error != nil {
		return fmt.Errorf("update page: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPageNotFound
	}
	return nil
}

// Delete 删除页面
func (r *PageRepository) Delete(id uuid.UUID) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("page repository is unavailable")
	}

	result := r.db.Delete(&model.Page{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete page: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPageNotFound
	}
	return nil
}

// ListNavPages 获取导航栏页面列表（已发布且 show_in_nav=true）
func (r *PageRepository) ListNavPages() ([]model.Page, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("page repository is unavailable")
	}

	var pages []model.Page
	if err := r.db.Where("status = ? AND show_in_nav = ? AND (published_at IS NULL OR published_at <= ?)",
		model.PageStatusPublished, true, time.Now().UTC()).
		Order("sort_order ASC").
		Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("list nav pages: %w", err)
	}

	return pages, nil
}
