package api

import (
	"net/http"

	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// SearchHandler 搜索 API
type SearchHandler struct {
	postService *service.PostService
}

func NewSearchHandler(postService *service.PostService) *SearchHandler {
	return &SearchHandler{postService: postService}
}

// Search 前台全文搜索
func (h *SearchHandler) Search(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "搜索关键词不能为空")
		return
	}

	result, err := h.postService.ListPublic(service.PostListParams{
		Page:     1,
		PageSize: 20,
		Keyword:  keyword,
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, result)
}
