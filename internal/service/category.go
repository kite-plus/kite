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
	ErrInvalidCategoryPayload = errors.New("invalid category payload")
	ErrDuplicateCategorySlug  = errors.New("duplicate category slug")
)

type CategoryListParams struct {
	Page     int
	PageSize int
	Keyword  string
}

type CreateCategoryInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type UpdateCategoryInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type PatchCategoryInput struct {
	Name *string `json:"name"`
	Slug *string `json:"slug"`
}

type CategoryListResult struct {
	Items      []model.Category `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

type CategoryService struct {
	repo *repo.CategoryRepository
}

func NewCategoryService(repo *repo.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) List(params CategoryListParams) (*CategoryListResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("category service is unavailable")
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

	items, total, err := s.repo.List(repo.CategoryListParams{
		Page:     params.Page,
		PageSize: params.PageSize,
		Keyword:  strings.TrimSpace(params.Keyword),
	})
	if err != nil {
		return nil, err
	}

	return &CategoryListResult{Items: items, Pagination: Pagination{Page: params.Page, PageSize: params.PageSize, Total: total}}, nil
}

func (s *CategoryService) GetByID(id string) (*model.Category, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid category id", ErrInvalidCategoryPayload)
	}
	return s.repo.GetByID(parsedID)
}

func (s *CategoryService) Create(input CreateCategoryInput) (*model.Category, error) {
	item := &model.Category{Name: strings.TrimSpace(input.Name), Slug: strings.TrimSpace(input.Slug)}
	if err := validateCategory(item); err != nil {
		return nil, err
	}
	if err := ensureCategorySlugAvailable(s.repo, item.Slug, uuid.Nil); err != nil {
		return nil, err
	}
	if err := s.repo.Create(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *CategoryService) Update(id string, input UpdateCategoryInput) (*model.Category, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid category id", ErrInvalidCategoryPayload)
	}
	item, err := s.repo.GetByID(parsedID)
	if err != nil {
		return nil, err
	}
	item.Name = strings.TrimSpace(input.Name)
	item.Slug = strings.TrimSpace(input.Slug)
	if err := validateCategory(item); err != nil {
		return nil, err
	}
	if err := ensureCategorySlugAvailable(s.repo, item.Slug, item.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *CategoryService) Patch(id string, input PatchCategoryInput) (*model.Category, error) {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid category id", ErrInvalidCategoryPayload)
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
	if err := validateCategory(item); err != nil {
		return nil, err
	}
	if err := ensureCategorySlugAvailable(s.repo, item.Slug, item.ID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *CategoryService) Delete(id string) error {
	parsedID, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("%w: invalid category id", ErrInvalidCategoryPayload)
	}
	return s.repo.Delete(parsedID)
}

func validateCategory(item *model.Category) error {
	if item == nil {
		return fmt.Errorf("%w: category is required", ErrInvalidCategoryPayload)
	}
	if item.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidCategoryPayload)
	}
	if item.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrInvalidCategoryPayload)
	}
	return nil
}

func ensureCategorySlugAvailable(categoryRepo *repo.CategoryRepository, slug string, currentID uuid.UUID) error {
	existing, err := categoryRepo.GetBySlug(slug)
	if err == nil && existing != nil && existing.ID != currentID {
		return ErrDuplicateCategorySlug
	}
	if err != nil && !errors.Is(err, repo.ErrCategoryNotFound) {
		return err
	}
	return nil
}
