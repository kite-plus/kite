package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

func (h *CategoryHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	result, err := h.categoryService.List(service.CategoryListParams{Page: page, PageSize: pageSize, Keyword: c.Query("keyword")})
	if err != nil {
		handleCategoryError(c, err)
		return
	}
	Success(c, result)
}

func (h *CategoryHandler) GetByID(c *gin.Context) {
	item, err := h.categoryService.GetByID(c.Param("id"))
	if err != nil {
		handleCategoryError(c, err)
		return
	}
	Success(c, item)
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var input service.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	item, err := h.categoryService.Create(input)
	if err != nil {
		handleCategoryError(c, err)
		return
	}
	Created(c, item)
}

func (h *CategoryHandler) Update(c *gin.Context) {
	var input service.UpdateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	item, err := h.categoryService.Update(c.Param("id"), input)
	if err != nil {
		handleCategoryError(c, err)
		return
	}
	Success(c, item)
}

func (h *CategoryHandler) Patch(c *gin.Context) {
	var input service.PatchCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	item, err := h.categoryService.Patch(c.Param("id"), input)
	if err != nil {
		handleCategoryError(c, err)
		return
	}
	Success(c, item)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	if err := h.categoryService.Delete(c.Param("id")); err != nil {
		handleCategoryError(c, err)
		return
	}
	Success(c, gin.H{"deleted": true})
}

func handleCategoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCategoryPayload):
		Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrDuplicateCategorySlug):
		Error(c, http.StatusConflict, http.StatusConflict, "duplicate category slug")
	case errors.Is(err, repo.ErrCategoryNotFound):
		Error(c, http.StatusNotFound, http.StatusNotFound, "resource not found")
	default:
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
