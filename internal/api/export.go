package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/amigoer/kite-blog/internal/model"
	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// ExportData 导出数据结构
type ExportData struct {
	ExportedAt  string              `json:"exported_at"`
	Version     string              `json:"version"`
	Posts       []model.Post        `json:"posts"`
	Pages       []model.Page        `json:"pages"`
	Categories  []model.Category    `json:"categories"`
	Tags        []model.Tag         `json:"tags"`
	Comments    []model.Comment     `json:"comments"`
	FriendLinks []model.FriendLink  `json:"friend_links"`
}

// ExportHandler 数据导出处理器
type ExportHandler struct {
	postService     *service.PostService
	pageService     *service.PageService
	friendLinkSvc   *service.FriendLinkService
	categoryRepo    *repo.CategoryRepository
	tagRepo         *repo.TagRepository
	commentRepo     *repo.CommentRepository
}

func NewExportHandler(
	postService *service.PostService,
	pageService *service.PageService,
	friendLinkSvc *service.FriendLinkService,
	categoryRepo *repo.CategoryRepository,
	tagRepo *repo.TagRepository,
	commentRepo *repo.CommentRepository,
) *ExportHandler {
	return &ExportHandler{
		postService:   postService,
		pageService:   pageService,
		friendLinkSvc: friendLinkSvc,
		categoryRepo:  categoryRepo,
		tagRepo:       tagRepo,
		commentRepo:   commentRepo,
	}
}

// Export 导出全站数据为 JSON
func (h *ExportHandler) Export(c *gin.Context) {
	data := ExportData{
		ExportedAt: time.Now().Format(time.RFC3339),
		Version:    "1.0",
	}

	// 导出所有文章（包括草稿）
	posts, _ := h.getAllPosts()
	data.Posts = posts

	// 导出所有页面
	if pages, err := h.pageService.ListAll(); err == nil {
		data.Pages = pages
	}

	// 导出所有分类
	if categories, _, err := h.categoryRepo.List(repo.CategoryListParams{Page: 1, PageSize: 1000}); err == nil {
		data.Categories = categories
	}

	// 导出所有标签
	if tags, _, err := h.tagRepo.List(repo.TagListParams{Page: 1, PageSize: 1000}); err == nil {
		data.Tags = tags
	}

	// 导出所有评论
	if comments, _, err := h.commentRepo.List(repo.CommentListParams{Page: 1, PageSize: 10000}); err == nil {
		data.Comments = comments
	}

	// 导出友链
	if result, err := h.friendLinkSvc.ListPublic(service.FriendLinkListParams{Page: 1, PageSize: 1000}); err == nil {
		data.FriendLinks = result.Items
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		Error(c, http.StatusInternalServerError, 500, "failed to export data")
		return
	}

	filename := fmt.Sprintf("kite-export-%s.json", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "application/json; charset=utf-8", jsonData)
}

func (h *ExportHandler) getAllPosts() ([]model.Post, error) {
	var allPosts []model.Post
	page := 1
	for {
		result, err := h.postService.List(service.PostListParams{Page: page, PageSize: 100})
		if err != nil {
			return allPosts, err
		}
		allPosts = append(allPosts, result.Items...)
		if len(allPosts) >= int(result.Pagination.Total) {
			break
		}
		page++
	}
	return allPosts, nil
}
