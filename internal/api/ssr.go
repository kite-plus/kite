package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
	"github.com/gin-gonic/gin"
)

// SSRHandler 前台 SSR 页面处理器
type SSRHandler struct {
	cfg             *config.Config
	postService     *service.PostService
	pageService     *service.PageService
	friendLinkSvc   *service.FriendLinkService
	categoryRepo    *repo.CategoryRepository
	tagRepo         *repo.TagRepository
	pageRepo        *repo.PageRepository
	postRepo        *repo.PostRepository
	settingsService *service.SettingsService
	slugHistoryRepo *repo.SlugHistoryRepository
}

func NewSSRHandler(
	cfg *config.Config,
	postService *service.PostService,
	pageService *service.PageService,
	friendLinkSvc *service.FriendLinkService,
	categoryRepo *repo.CategoryRepository,
	tagRepo *repo.TagRepository,
	pageRepo *repo.PageRepository,
	postRepo *repo.PostRepository,
	settingsService *service.SettingsService,
	slugHistoryRepo *repo.SlugHistoryRepository,
) *SSRHandler {
	return &SSRHandler{
		cfg:             cfg,
		postService:     postService,
		pageService:     pageService,
		friendLinkSvc:   friendLinkSvc,
		categoryRepo:    categoryRepo,
		tagRepo:         tagRepo,
		pageRepo:        pageRepo,
		postRepo:        postRepo,
		settingsService: settingsService,
		slugHistoryRepo: slugHistoryRepo,
	}
}

// commonData 构建所有页面共享的模板数据
func (h *SSRHandler) commonData(pageTitle string) gin.H {
	siteURL := strings.TrimRight(h.cfg.Site.SiteURL, "/")
	data := gin.H{
		"SiteName":    h.cfg.Site.SiteName,
		"SiteURL":     siteURL,
		"Description": h.cfg.Site.Description,
		"Keywords":    h.cfg.Site.Keywords,
		"Favicon":     h.cfg.Site.Favicon,
		"Logo":        h.cfg.Site.Logo,
		"ICP":         h.cfg.Site.ICP,
		"Footer":      h.cfg.Site.Footer,
		"PageTitle":   pageTitle,
	}

	// 优先使用自定义菜单，否则 fallback 到 NavPages
	if h.settingsService != nil {
		if menus := h.settingsService.GetNavMenus(); len(menus) > 0 {
			data["NavMenus"] = menus
			return data
		}
	}

	// Fallback：使用获取 NavPages 的方式
	if navPages, err := h.pageRepo.ListNavPages(); err == nil {
		data["NavPages"] = navPages
	}

	return data
}

// paginationData 构建分页模板数据
func paginationData(page, pageSize int, total int64, basePath string) gin.H {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	// 生成页码列表（最多显示 7 页）
	var pageNumbers []int
	start := page - 3
	end := page + 3
	if start < 1 {
		start = 1
		end = 7
	}
	if end > totalPages {
		end = totalPages
		start = end - 6
		if start < 1 {
			start = 1
		}
	}
	for i := start; i <= end; i++ {
		pageNumbers = append(pageNumbers, i)
	}

	return gin.H{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Total":       total,
		"PageSize":    pageSize,
		"BasePath":    basePath,
		"PageNumbers": pageNumbers,
	}
}

// Index 首页 — 文章列表 + 分页
func (h *SSRHandler) Index(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := h.cfg.Post.PostsPerPage
	if pageSize <= 0 {
		pageSize = 10
	}

	result, err := h.postService.ListPublic(service.PostListParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.renderError(c, http.StatusInternalServerError)
		return
	}

	data := h.commonData("")
	data["Posts"] = result.Items
	data["Pagination"] = paginationData(page, pageSize, result.Pagination.Total, "/")
	if page > 1 {
		data["CanonicalURL"] = data["SiteURL"].(string) + fmt.Sprintf("/?page=%d", page)
	} else {
		data["CanonicalURL"] = data["SiteURL"].(string) + "/"
	}

	c.HTML(http.StatusOK, "index.html", data)
}

// PostDetail 文章详情页
func (h *SSRHandler) PostDetail(c *gin.Context) {
	slug := c.Param("slug")
	post, err := h.postService.GetPublicBySlug(slug)
	if err != nil {
		// 查找 slug 历史记录，如存在则 301 重定向到新 slug
		if h.slugHistoryRepo != nil {
			if history, hErr := h.slugHistoryRepo.FindBySlug(slug); hErr == nil {
				newPost, pErr := h.postService.GetPublicByID(history.PostID.String())
				if pErr == nil {
					c.Redirect(http.StatusMovedPermanently, "/posts/"+newPost.Slug)
					return
				}
			}
		}
		h.renderError(c, http.StatusNotFound)
		return
	}

	// 自增浏览计数
	if h.postRepo != nil {
		_ = h.postRepo.IncrementViewCount(post.ID)
		post.ViewCount++
	}

	data := h.commonData(post.Title)
	data["CanonicalURL"] = data["SiteURL"].(string) + "/posts/" + post.Slug

	hasPassword := post.Password != ""
	hasProtected := service.HasProtectedBlocks(post.ContentHTML)

	if hasPassword {
		// 全局密码：隐藏全部内容，由前台密码表单解锁
		post.ContentHTML = ""
		post.ContentMarkdown = ""
		data["NeedPassword"] = true
	} else if hasProtected {
		// 片段保护：将 protected 块替换为占位符
		post.ContentHTML = service.FilterProtectedHTML(post.ContentHTML)
	}

	data["Post"] = post

	c.HTML(http.StatusOK, "post.html", data)
}

// CategoryArchive 分类归档页
func (h *SSRHandler) CategoryArchive(c *gin.Context) {
	slug := c.Param("slug")
	category, err := h.categoryRepo.GetBySlug(slug)
	if err != nil {
		h.renderError(c, http.StatusNotFound)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := h.cfg.Post.PostsPerPage
	if pageSize <= 0 {
		pageSize = 10
	}

	result, err := h.postService.ListPublic(service.PostListParams{
		Page:       page,
		PageSize:   pageSize,
		CategoryID: category.ID.String(),
	})
	if err != nil {
		h.renderError(c, http.StatusInternalServerError)
		return
	}

	data := h.commonData(category.Name)
	data["Posts"] = result.Items
	data["ArchiveType"] = "分类"
	data["ArchiveName"] = category.Name
	data["Pagination"] = paginationData(page, pageSize, result.Pagination.Total, "/categories/"+slug)
	canonicalPath := "/categories/" + slug
	if page > 1 {
		canonicalPath += fmt.Sprintf("?page=%d", page)
	}
	data["CanonicalURL"] = data["SiteURL"].(string) + canonicalPath

	c.HTML(http.StatusOK, "archive.html", data)
}

// TagArchive 标签归档页
func (h *SSRHandler) TagArchive(c *gin.Context) {
	slug := c.Param("slug")
	tag, err := h.tagRepo.GetBySlug(slug)
	if err != nil {
		h.renderError(c, http.StatusNotFound)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize := h.cfg.Post.PostsPerPage
	if pageSize <= 0 {
		pageSize = 10
	}

	result, err := h.postService.ListPublic(service.PostListParams{
		Page:     page,
		PageSize: pageSize,
		TagID:    tag.ID.String(),
	})
	if err != nil {
		h.renderError(c, http.StatusInternalServerError)
		return
	}

	data := h.commonData(tag.Name)
	data["Posts"] = result.Items
	data["ArchiveType"] = "标签"
	data["ArchiveName"] = tag.Name
	data["Pagination"] = paginationData(page, pageSize, result.Pagination.Total, "/tags/"+slug)
	canonicalPath := "/tags/" + slug
	if page > 1 {
		canonicalPath += fmt.Sprintf("?page=%d", page)
	}
	data["CanonicalURL"] = data["SiteURL"].(string) + canonicalPath

	c.HTML(http.StatusOK, "archive.html", data)
}

// PageDetail 独立页面（支持自定义模板）
func (h *SSRHandler) PageDetail(c *gin.Context) {
	slug := c.Param("slug")
	pg, err := h.pageService.GetPublicBySlug(slug)
	if err != nil {
		h.renderError(c, http.StatusNotFound)
		return
	}

	data := h.commonData(pg.Title)
	data["Page"] = pg
	data["CanonicalURL"] = data["SiteURL"].(string) + "/pages/" + pg.Slug

	// 解析页面 Config JSON 注入模板上下文
	pageConfig := make(map[string]interface{})
	if pg.Config != "" {
		_ = json.Unmarshal([]byte(pg.Config), &pageConfig)
	}
	data["PageConfig"] = pageConfig

	// 根据 Template 字段选择模板
	tmplName := "pages/default.html"
	if pg.Template != "" && pg.Template != "default" {
		tmplName = "pages/" + pg.Template + ".html"
	}

	c.HTML(http.StatusOK, tmplName, data)
}

// Friends 友情链接页
func (h *SSRHandler) Friends(c *gin.Context) {
	result, err := h.friendLinkSvc.ListPublic(service.FriendLinkListParams{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		h.renderError(c, http.StatusInternalServerError)
		return
	}

	data := h.commonData("友情链接")
	data["FriendLinks"] = result.Items

	c.HTML(http.StatusOK, "friends.html", data)
}

// renderError 渲染错误页面
func (h *SSRHandler) renderError(c *gin.Context, statusCode int) {
	data := h.commonData("")
	c.HTML(statusCode, "404.html", data)
}
