package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrPostNotFound = errors.New("post not found")

type PostListParams struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) List(params PostListParams) ([]model.Post, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("post repository is unavailable")
	}

	query := r.db.Model(&model.Post{})
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		query = query.Where("title LIKE ? OR summary LIKE ? OR content LIKE ?", keyword, keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
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

	var posts []model.Post
	if err := query.Order("created_at DESC").Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).Find(&posts).Error; err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}

	return posts, total, nil
}

func (r *PostRepository) GetByID(id uuid.UUID) (*model.Post, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("post repository is unavailable")
	}

	var post model.Post
	if err := r.db.First(&post, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("get post by id: %w", err)
	}

	return &post, nil
}

func (r *PostRepository) GetBySlug(slug string) (*model.Post, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("post repository is unavailable")
	}

	var post model.Post
	if err := r.db.First(&post, "slug = ?", slug).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("get post by slug: %w", err)
	}

	return &post, nil
}

func (r *PostRepository) Create(post *model.Post) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("post repository is unavailable")
	}
	if err := r.db.Create(post).Error; err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	return nil
}

func (r *PostRepository) Update(post *model.Post) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("post repository is unavailable")
	}
	result := r.db.Model(post).Select("title", "slug", "summary", "content", "status", "cover_image", "published_at").Updates(post)
	if result.Error != nil {
		return fmt.Errorf("update post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *PostRepository) Delete(id uuid.UUID) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("post repository is unavailable")
	}
	result := r.db.Delete(&model.Post{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete post: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}
