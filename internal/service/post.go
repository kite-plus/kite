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
	ErrInvalidPostPayload = errors.New("invalid post payload")
	ErrDuplicateSlug      = errors.New("duplicate slug")
)

type PostListParams struct {
	Page       int
	PageSize   int
	Status     string
	Keyword    string
	TagID      string
	CategoryID string
}

type CreatePostInput struct {
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Summary         string     `json:"summary"`
	ContentMarkdown string     `json:"content_markdown"`
	ContentHTML     string     `json:"content_html"`
	Status          string     `json:"status"`
	CoverImage      string     `json:"cover_image"`
	Password        string     `json:"password"`
	PublishedAt     *time.Time `json:"published_at"`
	ShowComments    *bool      `json:"show_comments"`
	CategoryID      *string    `json:"category_id"`
	TagIDs          []string   `json:"tag_ids"`
}

type UpdatePostInput struct {
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Summary         string     `json:"summary"`
	ContentMarkdown string     `json:"content_markdown"`
	ContentHTML     string     `json:"content_html"`
	Status          string     `json:"status"`
	CoverImage      string     `json:"cover_image"`
	Password        string     `json:"password"`
	PublishedAt     *time.Time `json:"published_at"`
	ShowComments    *bool      `json:"show_comments"`
	CategoryID      *string    `json:"category_id"`
	TagIDs          []string   `json:"tag_ids"`
}

type PatchPostInput struct {
	Title           *string    `json:"title"`
	Slug            *string    `json:"slug"`
	Summary         *string    `json:"summary"`
	ContentMarkdown *string    `json:"content_markdown"`
	ContentHTML     *string    `json:"content_html"`
	Status          *string    `json:"status"`
	CoverImage      *string    `json:"cover_image"`
	Password        *string    `json:"password"`
	PublishedAt     *time.Time `json:"published_at"`
	ShowComments    *bool      `json:"show_comments"`
	CategoryID      *string    `json:"category_id"`
	TagIDs          *[]string  `json:"tag_ids"`
}

type PostListResult struct {
	Items      []model.Post `json:"items"`
	Pagination Pagination   `json:"pagination"`
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type PostService struct {
	postRepo     *repo.PostRepository
	tagRepo      *repo.TagRepository
	categoryRepo *repo.CategoryRepository
}

func NewPostService(postRepo *repo.PostRepository, tagRepo *repo.TagRepository, categoryRepo *repo.CategoryRepository) *PostService {
	return &PostService{postRepo: postRepo, tagRepo: tagRepo, categoryRepo: categoryRepo}
}

func (s *PostService) List(params PostListParams) (*PostListResult, error) {
	return s.list(params, false)
}

func (s *PostService) ListPublic(params PostListParams) (*PostListResult, error) {
	return s.list(params, true)
}

func (s *PostService) list(params PostListParams, publicOnly bool) (*PostListResult, error) {
	if s == nil || s.postRepo == nil {
		return nil, fmt.Errorf("post service is unavailable")
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

	tagID, err := parseOptionalUUID(params.TagID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tag_id", ErrInvalidPostPayload)
	}
	categoryID, err := parseOptionalUUID(params.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid category_id", ErrInvalidPostPayload)
	}

	posts, total, err := s.postRepo.List(repo.PostListParams{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Status:     normalizeListStatus(params.Status, publicOnly),
		Keyword:    strings.TrimSpace(params.Keyword),
		TagID:      tagID,
		CategoryID: categoryID,
		PublicOnly: publicOnly,
	})
	if err != nil {
		return nil, err
	}

	return &PostListResult{
		Items: posts,
		Pagination: Pagination{
			Page:     params.Page,
			PageSize: params.PageSize,
			Total:    total,
		},
	}, nil
}

func (s *PostService) GetByID(id string) (*model.Post, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetByID(parsedID)
}

func (s *PostService) GetPublicByID(id string) (*model.Post, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetPublicByID(parsedID)
}

func (s *PostService) GetBySlug(slug string) (*model.Post, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrInvalidPostPayload)
	}
	return s.postRepo.GetBySlug(slug)
}

func (s *PostService) GetPublicBySlug(slug string) (*model.Post, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrInvalidPostPayload)
	}
	return s.postRepo.GetPublicBySlug(slug)
}

func (s *PostService) Create(input CreatePostInput) (*model.Post, error) {
	categoryID, tags, err := s.resolvePostRelations(input.CategoryID, input.TagIDs)
	if err != nil {
		return nil, err
	}

	post := &model.Post{
		Title:           strings.TrimSpace(input.Title),
		Slug:            strings.TrimSpace(input.Slug),
		Summary:         strings.TrimSpace(input.Summary),
		ContentMarkdown: strings.TrimSpace(input.ContentMarkdown),
		ContentHTML:     input.ContentHTML,
		Status:          normalizeStatus(input.Status),
		CoverImage:      strings.TrimSpace(input.CoverImage),
		Password:        input.Password,
		PublishedAt:     normalizeTimePointer(input.PublishedAt),
		ShowComments:    normalizeShowComments(input.ShowComments, true),
		CategoryID:      categoryID,
		Tags:            tags,
	}

	if err := preparePostForSave(post); err != nil {
		return nil, err
	}
	if err := validatePost(post); err != nil {
		return nil, err
	}
	if err := ensureSlugAvailable(s.postRepo, post.Slug, uuid.Nil); err != nil {
		return nil, err
	}
	if err := s.postRepo.Create(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *PostService) Update(id string, input UpdatePostInput) (*model.Post, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	existing, err := s.postRepo.GetByID(parsedID)
	if err != nil {
		return nil, err
	}

	categoryID, tags, err := s.resolvePostRelations(input.CategoryID, input.TagIDs)
	if err != nil {
		return nil, err
	}

	existing.Title = strings.TrimSpace(input.Title)
	existing.Slug = strings.TrimSpace(input.Slug)
	existing.Summary = strings.TrimSpace(input.Summary)
	existing.ContentMarkdown = strings.TrimSpace(input.ContentMarkdown)
	existing.ContentHTML = input.ContentHTML
	existing.Status = normalizeStatus(input.Status)
	existing.CoverImage = strings.TrimSpace(input.CoverImage)
	existing.Password = input.Password
	if input.PublishedAt != nil {
		existing.PublishedAt = normalizeTimePointer(input.PublishedAt)
	}
	existing.ShowComments = normalizeShowComments(input.ShowComments, existing.ShowComments)
	existing.CategoryID = categoryID
	existing.Tags = tags

	if err := preparePostForSave(existing); err != nil {
		return nil, err
	}
	if err := validatePost(existing); err != nil {
		return nil, err
	}
	if err := ensureSlugAvailable(s.postRepo, existing.Slug, existing.ID); err != nil {
		return nil, err
	}
	if err := s.postRepo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *PostService) Patch(id string, input PatchPostInput) (*model.Post, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	existing, err := s.postRepo.GetByID(parsedID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		existing.Title = strings.TrimSpace(*input.Title)
	}
	if input.Slug != nil {
		existing.Slug = strings.TrimSpace(*input.Slug)
	}
	if input.Summary != nil {
		existing.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.ContentMarkdown != nil {
		existing.ContentMarkdown = strings.TrimSpace(*input.ContentMarkdown)
	}
	if input.ContentHTML != nil {
		existing.ContentHTML = *input.ContentHTML
	}
	if input.Status != nil {
		existing.Status = normalizeStatus(*input.Status)
	}
	if input.CoverImage != nil {
		existing.CoverImage = strings.TrimSpace(*input.CoverImage)
	}
	if input.Password != nil {
		existing.Password = *input.Password
	}
	if input.PublishedAt != nil {
		existing.PublishedAt = normalizeTimePointer(input.PublishedAt)
	}
	if input.ShowComments != nil {
		existing.ShowComments = *input.ShowComments
	}
	if input.CategoryID != nil {
		categoryID, err := s.resolveCategoryID(input.CategoryID)
		if err != nil {
			return nil, err
		}
		existing.CategoryID = categoryID
	}
	if input.TagIDs != nil {
		tags, err := s.resolveTags(*input.TagIDs)
		if err != nil {
			return nil, err
		}
		existing.Tags = tags
	}

	if err := preparePostForSave(existing); err != nil {
		return nil, err
	}
	if err := validatePost(existing); err != nil {
		return nil, err
	}
	if err := ensureSlugAvailable(s.postRepo, existing.Slug, existing.ID); err != nil {
		return nil, err
	}
	if err := s.postRepo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *PostService) Delete(id string) error {
	parsedID, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.postRepo.Delete(parsedID)
}

func (s *PostService) resolvePostRelations(categoryIDInput *string, tagIDs []string) (*uuid.UUID, []model.Tag, error) {
	categoryID, err := s.resolveCategoryID(categoryIDInput)
	if err != nil {
		return nil, nil, err
	}
	tags, err := s.resolveTags(tagIDs)
	if err != nil {
		return nil, nil, err
	}
	return categoryID, tags, nil
}

func (s *PostService) resolveCategoryID(categoryIDInput *string) (*uuid.UUID, error) {
	if categoryIDInput == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*categoryIDInput)
	if trimmed == "" {
		return nil, nil
	}
	parsedID, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid category_id", ErrInvalidPostPayload)
	}
	if s.categoryRepo != nil {
		if _, err := s.categoryRepo.GetByID(parsedID); err != nil {
			if errors.Is(err, repo.ErrCategoryNotFound) {
				return nil, fmt.Errorf("%w: category not found", ErrInvalidPostPayload)
			}
			return nil, err
		}
	}
	return &parsedID, nil
}

func (s *PostService) resolveTags(tagIDs []string) ([]model.Tag, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	if s.tagRepo == nil {
		return nil, fmt.Errorf("tag repository is unavailable")
	}

	seen := map[uuid.UUID]struct{}{}
	tags := make([]model.Tag, 0, len(tagIDs))
	for _, rawID := range tagIDs {
		parsedID, err := uuid.Parse(strings.TrimSpace(rawID))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid tag_ids value", ErrInvalidPostPayload)
		}
		if _, ok := seen[parsedID]; ok {
			continue
		}
		tag, err := s.tagRepo.GetByID(parsedID)
		if err != nil {
			if errors.Is(err, repo.ErrTagNotFound) {
				return nil, fmt.Errorf("%w: tag not found", ErrInvalidPostPayload)
			}
			return nil, err
		}
		seen[parsedID] = struct{}{}
		tags = append(tags, *tag)
	}
	return tags, nil
}

func parseUUID(id string) (uuid.UUID, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid post id", ErrInvalidPostPayload)
	}
	return parsedID, nil
}

func parseOptionalUUID(id string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, nil
	}
	parsedID, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	return &parsedID, nil
}

func validatePost(post *model.Post) error {
	if post == nil {
		return fmt.Errorf("%w: post is required", ErrInvalidPostPayload)
	}
	if post.Title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidPostPayload)
	}
	if post.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidPostPayload)
	}
	// 内容可以是 Markdown 或 HTML 任意一个非空
	if post.ContentMarkdown == "" && post.ContentHTML == "" {
		return fmt.Errorf("%w: content is required", ErrInvalidPostPayload)
	}
	if !isValidStatus(post.Status) {
		return fmt.Errorf("%w: invalid status", ErrInvalidPostPayload)
	}
	return nil
}

func ensureSlugAvailable(postRepo *repo.PostRepository, slug string, currentID uuid.UUID) error {
	existing, err := postRepo.GetBySlug(slug)
	if err == nil && existing != nil && existing.ID != currentID {
		return ErrDuplicateSlug
	}
	if err != nil && !errors.Is(err, repo.ErrPostNotFound) {
		return err
	}
	return nil
}

func preparePostForSave(post *model.Post) error {
	if post == nil {
		return nil
	}

	// 如果前端已传 ContentHTML，直接使用；否则从 Markdown 渲染
	if post.ContentHTML == "" && post.ContentMarkdown != "" {
		contentHTML, err := renderPostMarkdown(post.ContentMarkdown)
		if err != nil {
			return fmt.Errorf("%w: invalid markdown content", ErrInvalidPostPayload)
		}
		post.ContentHTML = contentHTML
	}

	if post.Status == model.PostStatusPublished && post.PublishedAt == nil {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	return nil
}

func normalizeTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func normalizeShowComments(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func normalizeListStatus(status string, publicOnly bool) string {
	if publicOnly {
		return ""
	}
	return strings.TrimSpace(status)
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return model.PostStatusDraft
	}
	return status
}

func isValidStatus(status string) bool {
	switch status {
	case model.PostStatusDraft, model.PostStatusPublished, model.PostStatusArchived:
		return true
	default:
		return false
	}
}
