package repo

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrCommentNotFound = errors.New("comment not found")

// CommentListParams 评论列表查询参数
type CommentListParams struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
	PostID   *uuid.UUID
}

// CommentRepository 评论数据仓库
type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// List 查询评论列表
func (r *CommentRepository) List(params CommentListParams) ([]model.Comment, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("comment repository is unavailable")
	}

	query := r.db.Model(&model.Comment{})

	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.PostID != nil {
		query = query.Where("post_id = ?", *params.PostID)
	}
	if params.Keyword != "" {
		keyword := "%" + strings.TrimSpace(params.Keyword) + "%"
		query = query.Where("content LIKE ? OR author LIKE ?", keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
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

	var comments []model.Comment
	if err := query.Preload("Post").Order("created_at DESC").
		Offset((params.Page - 1) * params.PageSize).Limit(params.PageSize).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("list comments: %w", err)
	}

	return comments, total, nil
}

// ListPublicByPostID 查询指定文章的已审核评论（树形结构：仅查顶层，预加载子回复）
func (r *CommentRepository) ListPublicByPostID(postID uuid.UUID, page, pageSize int) ([]model.Comment, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("comment repository is unavailable")
	}

	// 只统计和分页顶层评论（parent_id IS NULL）
	query := r.db.Model(&model.Comment{}).
		Where("post_id = ? AND status = ? AND parent_id IS NULL", postID, model.CommentStatusApproved)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count public comments: %w", err)
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var comments []model.Comment
	if err := r.db.Where("post_id = ? AND status = ? AND parent_id IS NULL", postID, model.CommentStatusApproved).
		Preload("Replies", "status = ?", model.CommentStatusApproved).
		Order("created_at ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("list public comments: %w", err)
	}

	return comments, total, nil
}

// GetByID 根据 ID 获取评论
func (r *CommentRepository) GetByID(id uuid.UUID) (*model.Comment, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("comment repository is unavailable")
	}

	var comment model.Comment
	if err := r.db.Preload("Post").First(&comment, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("get comment by id: %w", err)
	}

	return &comment, nil
}

// Create 创建评论
func (r *CommentRepository) Create(comment *model.Comment) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("comment repository is unavailable")
	}

	if err := r.db.Create(comment).Error; err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

// Update 更新评论
func (r *CommentRepository) Update(comment *model.Comment) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("comment repository is unavailable")
	}

	result := r.db.Model(comment).Select("status").Updates(comment)
	if result.Error != nil {
		return fmt.Errorf("update comment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

// Delete 删除评论
func (r *CommentRepository) Delete(id uuid.UUID) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("comment repository is unavailable")
	}

	result := r.db.Delete(&model.Comment{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete comment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

// CountByStatus 按状态统计评论数量
func (r *CommentRepository) CountByStatus() (map[string]int64, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("comment repository is unavailable")
	}

	type statusCount struct {
		Status string
		Count  int64
	}

	var results []statusCount
	if err := r.db.Model(&model.Comment{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("count comments by status: %w", err)
	}

	counts := map[string]int64{
		model.CommentStatusApproved: 0,
		model.CommentStatusPending:  0,
		model.CommentStatusSpam:     0,
	}
	var total int64
	for _, r := range results {
		counts[r.Status] = r.Count
		total += r.Count
	}
	counts["total"] = total

	return counts, nil
}
