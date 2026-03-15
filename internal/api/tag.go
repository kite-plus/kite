package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	tagService *service.TagService
}

func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

func (h *TagHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	result, err := h.tagService.List(service.TagListParams{Page: page, PageSize: pageSize, Keyword: c.Query("keyword")})
	if err != nil {
		handleTagError(c, err)
		return
	}
	Success(c, result)
}

func (h *TagHandler) GetByID(c *gin.Context) {
	item, err := h.tagService.GetByID(c.Param("id"))
	if err != nil {
		handleTagError(c, err)
		return
	}
	Success(c, item)
}

func (h *TagHandler) Create(c *gin.Context) {
	var input service.CreateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	item, err := h.tagService.Create(input)
	if err != nil {
		handleTagError(c, err)
		return
	}
	Created(c, item)
}

func (h *TagHandler) Update(c *gin.Context) {
	var input service.UpdateTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	item, err := h.tagService.Update(c.Param("id"), input)
	if err != nil {
		handleTagError(c, err)
		return
	}
	Success(c, item)
}

func (h *TagHandler) Patch(c *gin.Context) {
	var input service.PatchTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}
	item, err := h.tagService.Patch(c.Param("id"), input)
	if err != nil {
		handleTagError(c, err)
		return
	}
	Success(c, item)
}

func (h *TagHandler) Delete(c *gin.Context) {
	if err := h.tagService.Delete(c.Param("id")); err != nil {
		handleTagError(c, err)
		return
	}
	Success(c, gin.H{"deleted": true})
}

func handleTagError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidTagPayload):
		Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrDuplicateTagSlug):
		Error(c, http.StatusConflict, http.StatusConflict, "duplicate tag slug")
	case errors.Is(err, repo.ErrTagNotFound):
		Error(c, http.StatusNotFound, http.StatusNotFound, "resource not found")
	default:
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
