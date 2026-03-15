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
	ErrInvalidTagPayload = errors.New("invalid tag payload")
	ErrDuplicateTagSlug  = errors.New("duplicate tag slug")
)

type TagListParams struct {
	Page     int
	PageSize int
	Keyword  string
}

type CreateTagInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type UpdateTagInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type PatchTagInput struct {
	Name *string `json:"name"`
	Slug *string `json:"slug"`
}

type TagListResult struct {
	Items      []model.Tag `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

type TagService struct {
	repo *repo.TagRepository
}

func NewTagService(repo *repo.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) List(params TagListParams) (*TagListResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("tag service is unavailable")
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

	items, total, err := s.repo.List(repo.TagListParams{
		Page:     params.Page,
		PageSize: params.PageSize,
		Keyword:  strings.TrimSpace(params.Keyword),
	})
	if err != nil {
		return nil, err
	}

	return &TagListResult{Items: items, Pagination: Pagination{Page: params.Page, PageSize: params.PageSize, Total: total}}, nil
}

func (s *TagService) GetByID(id string) (*model.Tag, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tag id", ErrInvalidTagPayload)
	}
	return s.repo.GetByID(parsedID)
}

func (s *TagService) Create(input CreateTagInput) (*model.Tag, error) {
	item := &model.Tag{Name: strings.TrimSpace(input.Name), Slug: strings.TrimSpace(input.Slug)}
	if err := validateTag(item); err != nil {
		return nil, err
	}
	if err := ensureTagSlugAvailable(s.repo, item.Slug, uuid.Nil); err != nil {
		return nil, err
	}
	if err := s.repo.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TagService) Update(id string, input UpdateTagInput) (*model.Tag, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tag id", ErrInvalidTagPayload)
	}
	item, err := s.repo.GetByID(parsedID)
	if err != nil {
		return nil, err
	}
	item.Name = strings.TrimSpace(input.Name)
	item.Slug = strings.TrimSpace(input.Slug)
	if err := validateTag(item); err != nil {
		return nil, err
	}
	if err := ensureTagSlugAvailable(s.repo, item.Slug, item.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TagService) Patch(id string, input PatchTagInput) (*model.Tag, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tag id", ErrInvalidTagPayload)
	}
	item, err := s.repo.GetByID(parsedID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		item.Name = strings.TrimSpace(*input.Name)
	}
	if input.Slug != nil {
		item.Slug = strings.TrimSpace(*input.Slug)
	}
	if err := validateTag(item); err != nil {
		return nil, err
	}
	if err := ensureTagSlugAvailable(s.repo, item.Slug, item.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *TagService) Delete(id string) error {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("%w: invalid tag id", ErrInvalidTagPayload)
	}
	return s.repo.Delete(parsedID)
}

func validateTag(item *model.Tag) error {
	if item == nil {
		return fmt.Errorf("%w: tag is required", ErrInvalidTagPayload)
	}
	if item.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidTagPayload)
	}
	if item.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidTagPayload)
	}
	return nil
}

func ensureTagSlugAvailable(tagRepo *repo.TagRepository, slug string, currentID uuid.UUID) error {
	existing, err := tagRepo.GetBySlug(slug)
	if err == nil && existing != nil && existing.ID != currentID {
		return ErrDuplicateTagSlug
	}
	if err != nil && !errors.Is(err, repo.ErrTagNotFound) {
		return err
	}
	return nil
}
