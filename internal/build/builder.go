package build

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/model"
	"github.com/amigoer/kite-blog/internal/repo"
	"github.com/amigoer/kite-blog/internal/service"
)

// StaticBuilder 静态站点生成器
type StaticBuilder struct {
	cfg           *config.Config
	outputDir     string
	tmpl          *template.Template
	templateFS    fs.FS
	postService   *service.PostService
	pageService   *service.PageService
	friendLinkSvc *service.FriendLinkService
	categoryRepo  *repo.CategoryRepository
	tagRepo       *repo.TagRepository
	pageRepo      *repo.PageRepository
}

// New 创建静态生成器实例
func New(cfg *config.Config, templateFS fs.FS, outputDir string,
	postService *service.PostService,
	pageService *service.PageService,
	friendLinkSvc *service.FriendLinkService,
	categoryRepo *repo.CategoryRepository,
	tagRepo *repo.TagRepository,
	pageRepo *repo.PageRepository,
) *StaticBuilder {
	return &StaticBuilder{
		cfg:           cfg,
		outputDir:     outputDir,
		templateFS:    templateFS,
		postService:   postService,
		pageService:   pageService,
		friendLinkSvc: friendLinkSvc,
		categoryRepo:  categoryRepo,
		tagRepo:       tagRepo,
		pageRepo:      pageRepo,
	}
}

// Build 执行静态站点生成
func (b *StaticBuilder) Build() error {
	start := time.Now()
	log.Println("🪁 开始生成静态站点...")

	// 加载模板
	tmpl, err := b.loadTemplates()
	if err != nil {
		return fmt.Errorf("加载模板失败: %w", err)
	}
	b.tmpl = tmpl

	// 清理输出目录
	if err := os.RemoveAll(b.outputDir); err != nil {
		return fmt.Errorf("清理输出目录失败: %w", err)
	}
	if err := os.MkdirAll(b.outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 按顺序生成各类页面
	steps := []struct {
		name string
		fn   func() error
	}{
		{"首页", b.buildIndex},
		{"文章详情", b.buildPosts},
		{"分类归档", b.buildCategories},
		{"标签归档", b.buildTags},
		{"独立页面", b.buildPages},
		{"友链页", b.buildFriends},
		{"404 页", b.build404},
		{"RSS", b.buildRSS},
		{"Sitemap", b.buildSitemap},
		{"静态资源", b.copyStatic},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("生成%s失败: %w", step.name, err)
		}
	}

	elapsed := time.Since(start)
	log.Printf("✅ 静态站点生成完成，输出目录: %s（耗时 %s）", b.outputDir, elapsed.Round(time.Millisecond))
	return nil
}

// ─── 模板加载 ───

// templateFuncs 自定义模板函数（与 api/template.go 保持一致）
var templateFuncs = template.FuncMap{
	"formatDate": func(t interface{}) string {
		switch v := t.(type) {
		case time.Time:
			return v.Format("2006-01-02")
		case *time.Time:
			if v == nil {
				return ""
			}
			return v.Format("2006-01-02")
		default:
			return ""
		}
	},
	"safeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"add":         func(a, b int) int { return a + b },
	"subtract":    func(a, b int) int { return a - b },
	"currentYear": func() int { return time.Now().Year() },
	// iconifyURL 将 Iconify icon ID（如 "lucide:home"）转换为 SVG API URL
	"iconifyURL": func(icon string) string {
		if icon == "" {
			return ""
		}
		path := strings.Replace(icon, ":", "/", 1)
		return "https://api.iconify.design/" + path + ".svg"
	},
}

func (b *StaticBuilder) loadTemplates() (*template.Template, error) {
	return template.New("").Funcs(templateFuncs).ParseFS(
		b.templateFS,
		"templates/*.html",
		"templates/partials/*.html",
		"templates/pages/*.html",
	)
}

// ─── 公共数据构建 ───

func (b *StaticBuilder) commonData(pageTitle string) map[string]interface{} {
	data := map[string]interface{}{
		"SiteName":    b.cfg.Site.SiteName,
		"Description": b.cfg.Site.Description,
		"Keywords":    b.cfg.Site.Keywords,
		"Favicon":     b.cfg.Site.Favicon,
		"Logo":        b.cfg.Site.Logo,
		"ICP":         b.cfg.Site.ICP,
		"Footer":      b.cfg.Site.Footer,
		"PageTitle":   pageTitle,
	}
	if navPages, err := b.pageRepo.ListNavPages(); err == nil {
		data["NavPages"] = navPages
	}
	return data
}

func paginationData(page, pageSize int, total int64, basePath string) map[string]interface{} {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

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

	return map[string]interface{}{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Total":       total,
		"PageSize":    pageSize,
		"BasePath":    basePath,
		"PageNumbers": pageNumbers,
	}
}

// ─── 渲染工具 ───

// renderToFile 将模板渲染到文件
func (b *StaticBuilder) renderToFile(tmplName, filePath string, data map[string]interface{}) error {
	absPath := filepath.Join(b.outputDir, filePath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	var buf bytes.Buffer
	if err := b.tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return fmt.Errorf("渲染模板 %s 失败: %w", tmplName, err)
	}

	if err := os.WriteFile(absPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("写入文件 %s 失败: %w", absPath, err)
	}

	return nil
}

// writeFile 写入内容到文件
func (b *StaticBuilder) writeFile(filePath string, content []byte) error {
	absPath := filepath.Join(b.outputDir, filePath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return os.WriteFile(absPath, content, 0644)
}

// ─── 首页生成（含分页） ───

func (b *StaticBuilder) buildIndex() error {
	pageSize := b.cfg.Post.PostsPerPage
	if pageSize <= 0 {
		pageSize = 10
	}

	// 获取总数以计算页数
	firstPage, err := b.postService.ListPublic(service.PostListParams{
		Page:     1,
		PageSize: pageSize,
	})
	if err != nil {
		return err
	}

	totalPages := int(math.Ceil(float64(firstPage.Pagination.Total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	for page := 1; page <= totalPages; page++ {
		result, err := b.postService.ListPublic(service.PostListParams{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return err
		}

		data := b.commonData("")
		data["Posts"] = result.Items
		data["Pagination"] = paginationData(page, pageSize, result.Pagination.Total, "/")

		var filePath string
		if page == 1 {
			filePath = "index.html"
		} else {
			filePath = fmt.Sprintf("page/%d/index.html", page)
		}

		if err := b.renderToFile("index.html", filePath, data); err != nil {
			return err
		}
	}

	log.Printf("  📄 首页: %d 页", totalPages)
	return nil
}

// ─── 文章详情页生成 ───

func (b *StaticBuilder) buildPosts() error {
	// 获取所有已发布文章
	allPosts, err := b.getAllPublicPosts()
	if err != nil {
		return err
	}

	count := 0
	for _, post := range allPosts {
		data := b.commonData(post.Title)
		data["Post"] = post

		filePath := fmt.Sprintf("posts/%s/index.html", post.Slug)
		if err := b.renderToFile("post.html", filePath, data); err != nil {
			return fmt.Errorf("生成文章 %s 失败: %w", post.Slug, err)
		}
		count++
	}

	log.Printf("  📝 文章: %d 篇", count)
	return nil
}

// ─── 分类归档页生成 ───

func (b *StaticBuilder) buildCategories() error {
	categories, _, err := b.categoryRepo.List(repo.CategoryListParams{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		return err
	}

	pageSize := b.cfg.Post.PostsPerPage
	if pageSize <= 0 {
		pageSize = 10
	}

	count := 0
	for _, category := range categories {
		// 获取该分类下的文章总数
		firstPage, err := b.postService.ListPublic(service.PostListParams{
			Page:       1,
			PageSize:   pageSize,
			CategoryID: category.ID.String(),
		})
		if err != nil {
			continue
		}

		totalPages := int(math.Ceil(float64(firstPage.Pagination.Total) / float64(pageSize)))
		if totalPages < 1 {
			totalPages = 1
		}

		for page := 1; page <= totalPages; page++ {
			result, err := b.postService.ListPublic(service.PostListParams{
				Page:       page,
				PageSize:   pageSize,
				CategoryID: category.ID.String(),
			})
			if err != nil {
				continue
			}

			data := b.commonData(category.Name)
			data["Posts"] = result.Items
			data["ArchiveType"] = "分类"
			data["ArchiveName"] = category.Name
			data["Pagination"] = paginationData(page, pageSize, result.Pagination.Total, "/categories/"+category.Slug)

			var filePath string
			if page == 1 {
				filePath = fmt.Sprintf("categories/%s/index.html", category.Slug)
			} else {
				filePath = fmt.Sprintf("categories/%s/page/%d/index.html", category.Slug, page)
			}

			if err := b.renderToFile("archive.html", filePath, data); err != nil {
				return err
			}
		}
		count++
	}

	log.Printf("  📂 分类: %d 个", count)
	return nil
}

// ─── 标签归档页生成 ───

func (b *StaticBuilder) buildTags() error {
	tags, _, err := b.tagRepo.List(repo.TagListParams{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		return err
	}

	pageSize := b.cfg.Post.PostsPerPage
	if pageSize <= 0 {
		pageSize = 10
	}

	count := 0
	for _, tag := range tags {
		firstPage, err := b.postService.ListPublic(service.PostListParams{
			Page:     1,
			PageSize: pageSize,
			TagID:    tag.ID.String(),
		})
		if err != nil {
			continue
		}

		totalPages := int(math.Ceil(float64(firstPage.Pagination.Total) / float64(pageSize)))
		if totalPages < 1 {
			totalPages = 1
		}

		for page := 1; page <= totalPages; page++ {
			result, err := b.postService.ListPublic(service.PostListParams{
				Page:     page,
				PageSize: pageSize,
				TagID:    tag.ID.String(),
			})
			if err != nil {
				continue
			}

			data := b.commonData(tag.Name)
			data["Posts"] = result.Items
			data["ArchiveType"] = "标签"
			data["ArchiveName"] = tag.Name
			data["Pagination"] = paginationData(page, pageSize, result.Pagination.Total, "/tags/"+tag.Slug)

			var filePath string
			if page == 1 {
				filePath = fmt.Sprintf("tags/%s/index.html", tag.Slug)
			} else {
				filePath = fmt.Sprintf("tags/%s/page/%d/index.html", tag.Slug, page)
			}

			if err := b.renderToFile("archive.html", filePath, data); err != nil {
				return err
			}
		}
		count++
	}

	log.Printf("  🏷️  标签: %d 个", count)
	return nil
}

// ─── 独立页面生成 ───

func (b *StaticBuilder) buildPages() error {
	result, err := b.pageService.ListPublic()
	if err != nil {
		return err
	}

	count := 0
	for _, pg := range result.Items {
		data := b.commonData(pg.Title)
		data["Page"] = pg

		// 解析页面 Config JSON
		pageConfig := make(map[string]interface{})
		if pg.Config != "" {
			_ = json.Unmarshal([]byte(pg.Config), &pageConfig)
		}
		data["PageConfig"] = pageConfig

		// 选择模板
		tmplName := "pages/default.html"
		if pg.Template != "" && pg.Template != "default" {
			tmplName = "pages/" + pg.Template + ".html"
		}

		filePath := fmt.Sprintf("pages/%s/index.html", pg.Slug)
		if err := b.renderToFile(tmplName, filePath, data); err != nil {
			return fmt.Errorf("生成页面 %s 失败: %w", pg.Slug, err)
		}
		count++
	}

	log.Printf("  📄 独立页面: %d 个", count)
	return nil
}

// ─── 友链页生成 ───

func (b *StaticBuilder) buildFriends() error {
	result, err := b.friendLinkSvc.ListPublic(service.FriendLinkListParams{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		return err
	}

	data := b.commonData("友情链接")
	data["FriendLinks"] = result.Items

	if err := b.renderToFile("friends.html", "friends/index.html", data); err != nil {
		return err
	}

	log.Printf("  🔗 友链: %d 个", len(result.Items))
	return nil
}

// ─── 404 页生成 ───

func (b *StaticBuilder) build404() error {
	data := b.commonData("")
	if err := b.renderToFile("404.html", "404.html", data); err != nil {
		return err
	}
	log.Println("  🚫 404 页")
	return nil
}

// ─── RSS 生成 ───

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	PubDate     string    `xml:"pubDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate,omitempty"`
	GUID        string `xml:"guid"`
}

func (b *StaticBuilder) buildRSS() error {
	siteURL := strings.TrimRight(b.cfg.Site.SiteURL, "/")

	result, err := b.postService.ListPublic(service.PostListParams{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		return err
	}

	items := make([]rssItem, 0, len(result.Items))
	var latestPubDate string
	for _, post := range result.Items {
		pubDate := ""
		if post.PublishedAt != nil {
			pubDate = post.PublishedAt.Format(time.RFC1123Z)
			if latestPubDate == "" {
				latestPubDate = pubDate
			}
		}
		desc := post.Summary
		if desc == "" && len(post.ContentHTML) > 300 {
			desc = post.ContentHTML[:300] + "..."
		} else if desc == "" {
			desc = post.ContentHTML
		}
		items = append(items, rssItem{
			Title:       post.Title,
			Link:        siteURL + "/posts/" + post.Slug,
			Description: desc,
			PubDate:     pubDate,
			GUID:        siteURL + "/posts/" + post.Slug,
		})
	}

	feed := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:       b.cfg.Site.SiteName,
			Link:        siteURL,
			Description: b.cfg.Site.Description,
			Language:    "zh-CN",
			PubDate:     latestPubDate,
			Items:       items,
		},
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return err
	}

	if err := b.writeFile("feed.xml", []byte(xml.Header+string(output))); err != nil {
		return err
	}

	log.Println("  📡 RSS feed")
	return nil
}

// ─── Sitemap 生成 ───

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	Lastmod    string `xml:"lastmod,omitempty"`
	Changefreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func (b *StaticBuilder) buildSitemap() error {
	siteURL := strings.TrimRight(b.cfg.Site.SiteURL, "/")

	urls := []sitemapURL{
		{Loc: siteURL + "/", Changefreq: "daily", Priority: "1.0"},
	}

	// 所有已发布文章
	allPosts, err := b.getAllPublicPosts()
	if err == nil {
		for _, post := range allPosts {
			lastmod := post.UpdatedAt.Format("2006-01-02")
			urls = append(urls, sitemapURL{
				Loc:        siteURL + "/posts/" + post.Slug,
				Lastmod:    lastmod,
				Changefreq: "weekly",
				Priority:   "0.8",
			})
		}
	}

	// 所有已发布页面
	pages, _ := b.pageService.ListPublic()
	if pages != nil {
		for _, page := range pages.Items {
			lastmod := page.UpdatedAt.Format("2006-01-02")
			urls = append(urls, sitemapURL{
				Loc:        siteURL + "/pages/" + page.Slug,
				Lastmod:    lastmod,
				Changefreq: "monthly",
				Priority:   "0.6",
			})
		}
	}

	sitemap := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	output, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return err
	}

	if err := b.writeFile("sitemap.xml", []byte(xml.Header+string(output))); err != nil {
		return err
	}

	log.Println("  🗺️  Sitemap")
	return nil
}

// ─── 静态资源复制 ───

func (b *StaticBuilder) copyStatic() error {
	staticFS, err := fs.Sub(b.templateFS, "templates/static")
	if err != nil {
		log.Println("  ⚠️  未找到 templates/static 目录，跳过静态资源")
		return nil
	}

	count := 0
	err = fs.WalkDir(staticFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		srcFile, err := staticFS.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstPath := filepath.Join(b.outputDir, "static", path)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		count++
		return nil
	})

	if err != nil {
		return err
	}

	log.Printf("  🎨 静态资源: %d 个文件", count)
	return nil
}

// ─── 辅助方法 ───

// getAllPublicPosts 获取所有已发布文章（分页遍历）
func (b *StaticBuilder) getAllPublicPosts() ([]model.Post, error) {
	var allPosts []model.Post
	page := 1
	pageSize := 100

	for {
		result, err := b.postService.ListPublic(service.PostListParams{
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return nil, err
		}

		allPosts = append(allPosts, result.Items...)

		if len(allPosts) >= int(result.Pagination.Total) {
			break
		}
		page++
	}

	return allPosts, nil
}
