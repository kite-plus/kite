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
	Page     int
	PageSize int
	Status   string
	Keyword  string
}

type CreatePostInput struct {
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Summary     string     `json:"summary"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	CoverImage  string     `json:"cover_image"`
	PublishedAt *time.Time `json:"published_at"`
}

type UpdatePostInput struct {
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Summary     string     `json:"summary"`
	Content     string     `json:"content"`
	Status      string     `json:"status"`
	CoverImage  string     `json:"cover_image"`
	PublishedAt *time.Time `json:"published_at"`
}

type PatchPostInput struct {
	Title       *string    `json:"title"`
	Slug        *string    `json:"slug"`
	Summary     *string    `json:"summary"`
	Content     *string    `json:"content"`
	Status      *string    `json:"status"`
	CoverImage  *string    `json:"cover_image"`
	PublishedAt *time.Time `json:"published_at"`
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
	repo *repo.PostRepository
}

func NewPostService(repo *repo.PostRepository) *PostService {
	return &PostService{repo: repo}
}

func (s *PostService) List(params PostListParams) (*PostListResult, error) {
	if s == nil || s.repo == nil {
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

	posts, total, err := s.repo.List(repo.PostListParams{
		Page:     params.Page,
		PageSize: params.PageSize,
		Status:   strings.TrimSpace(params.Status),
		Keyword:  strings.TrimSpace(params.Keyword),
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
	return s.repo.GetByID(parsedID)
}

func (s *PostService) GetBySlug(slug string) (*model.Post, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrInvalidPostPayload)
	}
	return s.repo.GetBySlug(slug)
}

func (s *PostService) Create(input CreatePostInput) (*model.Post, error) {
	post := &model.Post{
		Title:       strings.TrimSpace(input.Title),
		Slug:        strings.TrimSpace(input.Slug),
		Summary:     strings.TrimSpace(input.Summary),
		Content:     input.Content,
		Status:      normalizeStatus(input.Status),
		CoverImage:  strings.TrimSpace(input.CoverImage),
		PublishedAt: input.PublishedAt,
	}

	if err := validatePost(post); err != nil {
		return nil, err
	}
	if err := ensureSlugAvailable(s.repo, post.Slug, uuid.Nil); err != nil {
		return nil, err
	}
	if err := s.repo.Create(post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *PostService) Update(id string, input UpdatePostInput) (*model.Post, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(parsedID)
	if err != nil {
		return nil, err
	}

	existing.Title = strings.TrimSpace(input.Title)
	existing.Slug = strings.TrimSpace(input.Slug)
	existing.Summary = strings.TrimSpace(input.Summary)
	existing.Content = input.Content
	existing.Status = normalizeStatus(input.Status)
	existing.CoverImage = strings.TrimSpace(input.CoverImage)
	existing.PublishedAt = input.PublishedAt

	if err := validatePost(existing); err != nil {
		return nil, err
	}
	if err := ensureSlugAvailable(s.repo, existing.Slug, existing.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *PostService) Patch(id string, input PatchPostInput) (*model.Post, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(parsedID)
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
	if input.Content != nil {
		existing.Content = *input.Content
	}
	if input.Status != nil {
		existing.Status = normalizeStatus(*input.Status)
	}
	if input.CoverImage != nil {
		existing.CoverImage = strings.TrimSpace(*input.CoverImage)
	}
	if input.PublishedAt != nil {
		existing.PublishedAt = input.PublishedAt
	}

	if err := validatePost(existing); err != nil {
		return nil, err
	}
	if err := ensureSlugAvailable(s.repo, existing.Slug, existing.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *PostService) Delete(id string) error {
	parsedID, err := parseUUID(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(parsedID)
}

func parseUUID(id string) (uuid.UUID, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid post id", ErrInvalidPostPayload)
	}
	return parsedID, nil
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
	if post.Content == "" {
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
