package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/google/uuid"
)

var (
	ErrInvalidCommentPayload = errors.New("invalid comment payload")
)

// CommentListParams 评论列表查询参数
type CommentListParams struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
	PostID   string
}

// CreateCommentInput 前台访客提交评论的输入
type CreateCommentInput struct {
	Author   string `json:"author"`
	Email    string `json:"email"`
	Content  string `json:"content"`
	ParentID string `json:"parent_id"`
}

// ModerateCommentInput 管理端审核评论的输入
type ModerateCommentInput struct {
	Status string `json:"status"`
}

// CommentListResult 评论列表结果
type CommentListResult struct {
	Items      []model.Comment `json:"items"`
	Pagination Pagination      `json:"pagination"`
}

// CommentStats 评论统计
type CommentStats struct {
	Total    int64 `json:"total"`
	Approved int64 `json:"approved"`
	Pending  int64 `json:"pending"`
	Spam     int64 `json:"spam"`
}

// CommentService 评论业务服务
type CommentService struct {
	commentRepo         *repo.CommentRepository
	postRepo            *repo.PostRepository
	notificationService *NotificationService
	spamChecker         *SpamChecker
}

func NewCommentService(commentRepo *repo.CommentRepository, postRepo *repo.PostRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		spamChecker: NewSpamChecker(),
	}
}

// SetNotificationService 注入通知服务（避免循环依赖）
func (s *CommentService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

// List 管理端查询评论列表
func (s *CommentService) List(params CommentListParams) (*CommentListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	postID, err := parseOptionalUUID(params.PostID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid post_id", ErrInvalidCommentPayload)
	}

	comments, total, err := s.commentRepo.List(repo.CommentListParams{
		Page:     params.Page,
		PageSize: params.PageSize,
		Status:   strings.TrimSpace(params.Status),
		Keyword:  strings.TrimSpace(params.Keyword),
		PostID:   postID,
	})
	if err != nil {
		return nil, err
	}

	return &CommentListResult{
		Items: comments,
		Pagination: Pagination{
			Page:     params.Page,
			PageSize: params.PageSize,
			Total:    total,
		},
	}, nil
}

// ListPublicByPostID 前台查询指定文章的已审核评论
func (s *CommentService) ListPublicByPostID(postIDStr string, page, pageSize int) (*CommentListResult, error) {
	postID, err := uuid.Parse(strings.TrimSpace(postIDStr))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid post_id", ErrInvalidCommentPayload)
	}

	comments, total, err := s.commentRepo.ListPublicByPostID(postID, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &CommentListResult{
		Items: comments,
		Pagination: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

// Create 前台创建评论
func (s *CommentService) Create(postIDStr string, input CreateCommentInput, ip, userAgent string) (*model.Comment, error) {
	postID, err := uuid.Parse(strings.TrimSpace(postIDStr))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid post_id", ErrInvalidCommentPayload)
	}

	// 验证文章是否存在
	var postTitle string
	if s.postRepo != nil {
		post, err := s.postRepo.GetPublicByID(postID)
		if err != nil {
			if errors.Is(err, repo.ErrPostNotFound) {
				return nil, fmt.Errorf("%w: post not found", ErrInvalidCommentPayload)
			}
			return nil, err
		}
		postTitle = post.Title
	}

	author := strings.TrimSpace(input.Author)
	email := strings.TrimSpace(input.Email)
	content := strings.TrimSpace(input.Content)

	if author == "" {
		return nil, fmt.Errorf("%w: author is required", ErrInvalidCommentPayload)
	}
	if email == "" {
		return nil, fmt.Errorf("%w: email is required", ErrInvalidCommentPayload)
	}
	if content == "" {
		return nil, fmt.Errorf("%w: content is required", ErrInvalidCommentPayload)
	}

	// 解析 parent_id
	var parentID *uuid.UUID
	if input.ParentID != "" {
		pid, err := uuid.Parse(strings.TrimSpace(input.ParentID))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid parent_id", ErrInvalidCommentPayload)
		}
		parentID = &pid
	}

	// 垃圾评论检查：命中则自动标记为 spam
	status := model.CommentStatusPending
	if s.spamChecker != nil && s.spamChecker.IsSpam(content) {
		status = model.CommentStatusSpam
	}

	comment := &model.Comment{
		PostID:    postID,
		ParentID:  parentID,
		Author:    author,
		Email:     email,
		Content:   content,
		Status:    status,
		IP:        ip,
		UserAgent: userAgent,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	// 创建通知
	if s.notificationService != nil {
		_ = s.notificationService.CreateFromComment(comment, postTitle)
	}

	return comment, nil
}

// Moderate 管理端审核评论
func (s *CommentService) Moderate(idStr string, input ModerateCommentInput) (*model.Comment, error) {
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid comment id", ErrInvalidCommentPayload)
	}

	status := strings.TrimSpace(input.Status)
	if !isValidCommentStatus(status) {
		return nil, fmt.Errorf("%w: invalid status, must be approved/pending/spam", ErrInvalidCommentPayload)
	}

	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	comment.Status = status
	if err := s.commentRepo.Update(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// Delete 管理端删除评论
func (s *CommentService) Delete(idStr string) error {
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return fmt.Errorf("%w: invalid comment id", ErrInvalidCommentPayload)
	}

	return s.commentRepo.Delete(id)
}

// Stats 评论统计
func (s *CommentService) Stats() (*CommentStats, error) {
	counts, err := s.commentRepo.CountByStatus()
	if err != nil {
		return nil, err
	}

	return &CommentStats{
		Total:    counts["total"],
		Approved: counts[model.CommentStatusApproved],
		Pending:  counts[model.CommentStatusPending],
		Spam:     counts[model.CommentStatusSpam],
	}, nil
}

func isValidCommentStatus(status string) bool {
	switch status {
	case model.CommentStatusApproved, model.CommentStatusPending, model.CommentStatusSpam:
		return true
	default:
		return false
	}
}
