package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/google/uuid"
)

var (
	ErrInvalidPagePayload = errors.New("invalid page payload")
	ErrDuplicatePageSlug  = errors.New("duplicate page slug")
)

// PageListParams 页面列表查询参数
type PageListParams struct {
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

// CreatePageInput 创建页面输入
type CreatePageInput struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	ContentMarkdown string `json:"content_markdown"`
	Status          string `json:"status"`
	SortOrder       int    `json:"sort_order"`
	ShowInNav       bool   `json:"show_in_nav"`
	Template        string `json:"template"`
	Config          string `json:"config"`
}

// UpdatePageInput 全量更新页面输入
type UpdatePageInput struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	ContentMarkdown string `json:"content_markdown"`
	Status          string `json:"status"`
	SortOrder       int    `json:"sort_order"`
	ShowInNav       bool   `json:"show_in_nav"`
	Template        string `json:"template"`
	Config          string `json:"config"`
}

// PatchPageInput 局部更新页面输入
type PatchPageInput struct {
	Title           *string `json:"title"`
	Slug            *string `json:"slug"`
	ContentMarkdown *string `json:"content_markdown"`
	Status          *string `json:"status"`
	SortOrder       *int    `json:"sort_order"`
	ShowInNav       *bool   `json:"show_in_nav"`
	Template        *string `json:"template"`
	Config          *string `json:"config"`
}

// PageListResult 页面列表结果
type PageListResult struct {
	Items      []model.Page `json:"items"`
	Pagination Pagination   `json:"pagination"`
}

// PageService 独立页面业务服务
type PageService struct {
	pageRepo *repo.PageRepository
}

func NewPageService(pageRepo *repo.PageRepository) *PageService {
	return &PageService{pageRepo: pageRepo}
}

// List 管理端查询页面列表
func (s *PageService) List(params PageListParams) (*PageListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	pages, total, err := s.pageRepo.List(repo.PageListParams{
		Page:     params.Page,
		PageSize: params.PageSize,
		Status:   strings.TrimSpace(params.Status),
		Keyword:  strings.TrimSpace(params.Keyword),
	})
	if err != nil {
		return nil, err
	}

	return &PageListResult{
		Items: pages,
		Pagination: Pagination{
			Page:     params.Page,
			PageSize: params.PageSize,
			Total:    total,
		},
	}, nil
}

// ListPublic 前台获取已发布页面列表
func (s *PageService) ListPublic() (*PageListResult, error) {
	pages, total, err := s.pageRepo.List(repo.PageListParams{
		Page:       1,
		PageSize:   100,
		PublicOnly: true,
	})
	if err != nil {
		return nil, err
	}

	return &PageListResult{
		Items: pages,
		Pagination: Pagination{
			Page:     1,
			PageSize: 100,
			Total:    total,
		},
	}, nil
}

// GetByID 管理端获取页面详情
func (s *PageService) GetByID(idStr string) (*model.Page, error) {
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid page id", ErrInvalidPagePayload)
	}

	return s.pageRepo.GetByID(id)
}

// GetPublicBySlug 前台根据 slug 获取已发布页面
func (s *PageService) GetPublicBySlug(slug string) (*model.Page, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrInvalidPagePayload)
	}

	return s.pageRepo.GetPublicBySlug(slug)
}

// Create 创建页面
func (s *PageService) Create(input CreatePageInput) (*model.Page, error) {
	page := &model.Page{
		Title:           strings.TrimSpace(input.Title),
		Slug:            strings.TrimSpace(input.Slug),
		ContentMarkdown: strings.TrimSpace(input.ContentMarkdown),
		Status:          normalizePageStatus(input.Status),
		SortOrder:       input.SortOrder,
		ShowInNav:       input.ShowInNav,
		Template:        normalizePageTemplate(input.Template),
		Config:          input.Config,
	}

	if err := preparePageForSave(page); err != nil {
		return nil, err
	}
	if err := validatePage(page); err != nil {
		return nil, err
	}
	if err := ensurePageSlugAvailable(s.pageRepo, page.Slug, uuid.Nil); err != nil {
		return nil, err
	}

	if err := s.pageRepo.Create(page); err != nil {
		return nil, err
	}

	return page, nil
}

// Update 全量更新页面
func (s *PageService) Update(idStr string, input UpdatePageInput) (*model.Page, error) {
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid page id", ErrInvalidPagePayload)
	}

	existing, err := s.pageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	existing.Title = strings.TrimSpace(input.Title)
	existing.Slug = strings.TrimSpace(input.Slug)
	existing.ContentMarkdown = strings.TrimSpace(input.ContentMarkdown)
	existing.Status = normalizePageStatus(input.Status)
	existing.SortOrder = input.SortOrder
	existing.ShowInNav = input.ShowInNav
	existing.Template = normalizePageTemplate(input.Template)
	existing.Config = input.Config

	if err := preparePageForSave(existing); err != nil {
		return nil, err
	}
	if err := validatePage(existing); err != nil {
		return nil, err
	}
	if err := ensurePageSlugAvailable(s.pageRepo, existing.Slug, existing.ID); err != nil {
		return nil, err
	}
	if err := s.pageRepo.Update(existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// Patch 局部更新页面
func (s *PageService) Patch(idStr string, input PatchPageInput) (*model.Page, error) {
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid page id", ErrInvalidPagePayload)
	}

	existing, err := s.pageRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		existing.Title = strings.TrimSpace(*input.Title)
	}
	if input.Slug != nil {
		existing.Slug = strings.TrimSpace(*input.Slug)
	}
	if input.ContentMarkdown != nil {
		existing.ContentMarkdown = strings.TrimSpace(*input.ContentMarkdown)
	}
	if input.Status != nil {
		existing.Status = normalizePageStatus(*input.Status)
	}
	if input.SortOrder != nil {
		existing.SortOrder = *input.SortOrder
	}
	if input.ShowInNav != nil {
		existing.ShowInNav = *input.ShowInNav
	}
	if input.Template != nil {
		existing.Template = normalizePageTemplate(*input.Template)
	}
	if input.Config != nil {
		existing.Config = *input.Config
	}

	if err := preparePageForSave(existing); err != nil {
		return nil, err
	}
	if err := validatePage(existing); err != nil {
		return nil, err
	}
	if err := ensurePageSlugAvailable(s.pageRepo, existing.Slug, existing.ID); err != nil {
		return nil, err
	}
	if err := s.pageRepo.Update(existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// Delete 删除页面
func (s *PageService) Delete(idStr string) error {
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return fmt.Errorf("%w: invalid page id", ErrInvalidPagePayload)
	}

	return s.pageRepo.Delete(id)
}

// preparePageForSave 保存前处理（渲染 Markdown、自动填充发布时间）
func preparePageForSave(page *model.Page) error {
	if page == nil {
		return nil
	}

	contentHTML, err := renderPostMarkdown(page.ContentMarkdown)
	if err != nil {
		return fmt.Errorf("%w: invalid markdown content", ErrInvalidPagePayload)
	}
	page.ContentHTML = contentHTML

	if page.Status == model.PageStatusPublished && page.PublishedAt == nil {
		now := time.Now().UTC()
		page.PublishedAt = &now
	}

	return nil
}

func validatePage(page *model.Page) error {
	if page == nil {
		return fmt.Errorf("%w: page is required", ErrInvalidPagePayload)
	}
	if page.Title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidPagePayload)
	}
	if page.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidPagePayload)
	}
	if !isValidPageStatus(page.Status) {
		return fmt.Errorf("%w: invalid status", ErrInvalidPagePayload)
	}
	return nil
}

func ensurePageSlugAvailable(pageRepo *repo.PageRepository, slug string, currentID uuid.UUID) error {
	existing, err := pageRepo.GetBySlug(slug)
	if err == nil && existing != nil && existing.ID != currentID {
		return ErrDuplicatePageSlug
	}
	if err != nil && !errors.Is(err, repo.ErrPageNotFound) {
		return err
	}
	return nil
}

func normalizePageTemplate(tmpl string) string {
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return "default"
	}
	return tmpl
}

func normalizePageStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return model.PageStatusDraft
	}
	return status
}

func isValidPageStatus(status string) bool {
	switch status {
	case model.PageStatusDraft, model.PageStatusPublished:
		return true
	default:
		return false
	}
}
