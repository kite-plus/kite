package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

func (h *PostHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := h.postService.List(service.PostListParams{
		Page:       page,
		PageSize:   pageSize,
		Status:     c.Query("status"),
		Keyword:    c.Query("keyword"),
		TagID:      c.Query("tag_id"),
		CategoryID: c.Query("category_id"),
	})
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, result)
}

func (h *PostHandler) ListPublic(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := h.postService.ListPublic(service.PostListParams{
		Page:       page,
		PageSize:   pageSize,
		Keyword:    c.Query("keyword"),
		TagID:      c.Query("tag_id"),
		CategoryID: c.Query("category_id"),
	})
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, result)
}

func (h *PostHandler) GetByID(c *gin.Context) {
	post, err := h.postService.GetByID(c.Param("id"))
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, post)
}

func (h *PostHandler) GetPublicByID(c *gin.Context) {
	post, err := h.postService.GetPublicByID(c.Param("id"))
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, post)
}

func (h *PostHandler) GetBySlug(c *gin.Context) {
	post, err := h.postService.GetBySlug(c.Param("slug"))
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, post)
}

func (h *PostHandler) GetPublicBySlug(c *gin.Context) {
	post, err := h.postService.GetPublicBySlug(c.Param("slug"))
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, post)
}

func (h *PostHandler) Create(c *gin.Context) {
	var input service.CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	post, err := h.postService.Create(input)
	if err != nil {
		handlePostError(c, err)
		return
	}

	Created(c, post)
}

func (h *PostHandler) Update(c *gin.Context) {
	var input service.UpdatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	post, err := h.postService.Update(c.Param("id"), input)
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, post)
}

func (h *PostHandler) Patch(c *gin.Context) {
	var input service.PatchPostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "invalid request payload")
		return
	}

	post, err := h.postService.Patch(c.Param("id"), input)
	if err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, post)
}

func (h *PostHandler) Delete(c *gin.Context) {
	if err := h.postService.Delete(c.Param("id")); err != nil {
		handlePostError(c, err)
		return
	}

	Success(c, gin.H{"deleted": true})
}

func handlePostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidPostPayload):
		Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrDuplicateSlug):
		Error(c, http.StatusConflict, http.StatusConflict, "duplicate slug")
	case errors.Is(err, repo.ErrPostNotFound):
		Error(c, http.StatusNotFound, http.StatusNotFound, "resource not found")
	default:
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
	}
}
