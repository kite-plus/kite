package api

import (
	"net/http"

	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// AIHandler AI 辅助 API
type AIHandler struct {
	aiService *service.AIService
}

func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// Summary 生成文章摘要
func (h *AIHandler) Summary(c *gin.Context) {
	var input service.SummaryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "请求参数错误")
		return
	}
	if input.Content == "" {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "内容不能为空")
		return
	}

	summary, err := h.aiService.GenerateSummary(input)
	if err != nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, gin.H{"summary": summary})
}

// Tags 推荐标签
func (h *AIHandler) Tags(c *gin.Context) {
	var input service.TagSuggestionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "请求参数错误")
		return
	}
	if input.Content == "" {
		Error(c, http.StatusBadRequest, http.StatusBadRequest, "内容不能为空")
		return
	}

	tags, err := h.aiService.SuggestTags(input)
	if err != nil {
		Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}

	Success(c, gin.H{"tags": tags})
}
