package service

import (
	"encoding/json"
	"fmt"

	"github.com/amigoer/kite-blog/internal/config"
	"github.com/amigoer/kite-blog/internal/repo"
)

// 设置存储的 key 常量
const (
	SettingKeySite       = "site"
	SettingKeyPost       = "post"
	SettingKeyRender     = "render"
	SettingKeyAI         = "ai"
	SettingKeyAdmin      = "admin"
	SettingKeyDBDriver   = "db_driver"
	SettingKeyDBPath     = "db_path"
	SettingKeyRenderMode = "render_mode"
	SettingKeyNavMenus   = "nav_menus"
)

// SiteSettings 站点基础设置
type SiteSettings struct {
	SiteName    string `json:"site_name"`
	SiteURL     string `json:"site_url"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Favicon     string `json:"favicon"`
	Logo        string `json:"logo"`
	ICP         string `json:"icp"`
	Footer      string `json:"footer"`
}

// PostSettingsResp 文章相关设置
type PostSettingsResp struct {
	PostsPerPage    int    `json:"posts_per_page"`
	EnableComment   bool   `json:"enable_comment"`
	EnableToc       bool   `json:"enable_toc"`
	SummaryLength   int    `json:"summary_length"`
	DefaultCoverURL string `json:"default_cover_url"`
}

// RenderSettingsResp 渲染模式设置
type RenderSettingsResp struct {
	RenderMode string `json:"render_mode"`
	APIPrefix  string `json:"api_prefix"`
	EnableCORS bool   `json:"enable_cors"`
}

// AISettings AI 集成设置
type AISettings struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	AutoSummary bool   `json:"auto_summary"`
	AutoTag     bool   `json:"auto_tag"`
}

// AllSettings 全部设置聚合
type AllSettings struct {
	Site   SiteSettings       `json:"site"`
	Post   PostSettingsResp   `json:"post"`
	Render RenderSettingsResp `json:"render"`
	AI     AISettings         `json:"ai"`
}

// SettingsService 设置服务（从 SQLite 读写）
type SettingsService struct {
	settingsRepo *repo.SettingsRepository
	cfg          *config.Config // 运行时缓存
}

func NewSettingsService(cfg *config.Config, settingsRepo *repo.SettingsRepository) *SettingsService {
	svc := &SettingsService{settingsRepo: settingsRepo, cfg: cfg}
	// 启动时从 DB 加载到内存缓存
	svc.loadFromDB()
	return svc
}

// loadFromDB 从 DB 加载所有设置到运行时 config 缓存
func (s *SettingsService) loadFromDB() {
	all := s.settingsRepo.GetAll()

	if v, ok := all[SettingKeySite]; ok {
		var site config.SiteConfig
		if json.Unmarshal([]byte(v), &site) == nil {
			s.cfg.Site = site
		}
	}
	if v, ok := all[SettingKeyPost]; ok {
		var post config.PostConfig
		if json.Unmarshal([]byte(v), &post) == nil {
			s.cfg.Post = post
		}
	}
	if v, ok := all[SettingKeyRender]; ok {
		var render struct {
			RenderMode string `json:"render_mode"`
		}
		if json.Unmarshal([]byte(v), &render) == nil && render.RenderMode != "" {
			s.cfg.RenderMode = render.RenderMode
		}
	}
	if v, ok := all[SettingKeyAI]; ok {
		var ai config.AIConfig
		if json.Unmarshal([]byte(v), &ai) == nil {
			s.cfg.AI = ai
		}
	}
	if v, ok := all[SettingKeyAdmin]; ok {
		var admin config.AdminConfig
		if json.Unmarshal([]byte(v), &admin) == nil {
			s.cfg.Admin = admin
		}
	}
}

// Get 获取当前全部设置
func (s *SettingsService) Get() *AllSettings {
	return &AllSettings{
		Site: SiteSettings{
			SiteName:    s.cfg.Site.SiteName,
			SiteURL:     s.cfg.Site.SiteURL,
			Description: s.cfg.Site.Description,
			Keywords:    s.cfg.Site.Keywords,
			Favicon:     s.cfg.Site.Favicon,
			Logo:        s.cfg.Site.Logo,
			ICP:         s.cfg.Site.ICP,
			Footer:      s.cfg.Site.Footer,
		},
		Post: PostSettingsResp{
			PostsPerPage:    s.cfg.Post.PostsPerPage,
			EnableComment:   s.cfg.Post.EnableComment,
			EnableToc:       s.cfg.Post.EnableToc,
			SummaryLength:   s.cfg.Post.SummaryLength,
			DefaultCoverURL: s.cfg.Post.DefaultCoverURL,
		},
		Render: RenderSettingsResp{
			RenderMode: s.cfg.RenderMode,
			APIPrefix:  "/api/v1",
			EnableCORS: true,
		},
		AI: AISettings{
			Enabled:     s.cfg.AI.Enabled,
			Provider:    s.cfg.AI.Provider,
			APIKey:      maskAPIKey(s.cfg.AI.APIKey),
			Model:       s.cfg.AI.Model,
			AutoSummary: s.cfg.AI.AutoSummary,
			AutoTag:     s.cfg.AI.AutoTag,
		},
	}
}

// Update 更新设置（内存缓存 + 持久化到 DB）
func (s *SettingsService) Update(input AllSettings) (*AllSettings, error) {
	// 更新内存缓存
	s.cfg.Site.SiteName = input.Site.SiteName
	s.cfg.Site.SiteURL = input.Site.SiteURL
	s.cfg.Site.Description = input.Site.Description
	s.cfg.Site.Keywords = input.Site.Keywords
	s.cfg.Site.Favicon = input.Site.Favicon
	s.cfg.Site.Logo = input.Site.Logo
	s.cfg.Site.ICP = input.Site.ICP
	s.cfg.Site.Footer = input.Site.Footer

	if input.Post.PostsPerPage > 0 {
		s.cfg.Post.PostsPerPage = input.Post.PostsPerPage
	}
	s.cfg.Post.EnableComment = input.Post.EnableComment
	s.cfg.Post.EnableToc = input.Post.EnableToc
	if input.Post.SummaryLength > 0 {
		s.cfg.Post.SummaryLength = input.Post.SummaryLength
	}
	s.cfg.Post.DefaultCoverURL = input.Post.DefaultCoverURL

	if input.Render.RenderMode == config.RenderModeClassic || input.Render.RenderMode == config.RenderModeHeadless {
		s.cfg.RenderMode = input.Render.RenderMode
	}

	s.cfg.AI.Enabled = input.AI.Enabled
	s.cfg.AI.Provider = input.AI.Provider
	if input.AI.APIKey != "" && input.AI.APIKey != maskAPIKey(s.cfg.AI.APIKey) {
		s.cfg.AI.APIKey = input.AI.APIKey
	}
	s.cfg.AI.Model = input.AI.Model
	s.cfg.AI.AutoSummary = input.AI.AutoSummary
	s.cfg.AI.AutoTag = input.AI.AutoTag

	// 持久化到 DB
	if err := s.saveToDB(); err != nil {
		return nil, fmt.Errorf("保存设置失败: %w", err)
	}

	return s.Get(), nil
}

// saveToDB 将当前内存配置写入 DB
func (s *SettingsService) saveToDB() error {
	kvs := make(map[string]string)

	siteJSON, _ := json.Marshal(s.cfg.Site)
	kvs[SettingKeySite] = string(siteJSON)

	postJSON, _ := json.Marshal(s.cfg.Post)
	kvs[SettingKeyPost] = string(postJSON)

	renderJSON, _ := json.Marshal(map[string]string{"render_mode": s.cfg.RenderMode})
	kvs[SettingKeyRender] = string(renderJSON)

	aiJSON, _ := json.Marshal(s.cfg.AI)
	kvs[SettingKeyAI] = string(aiJSON)

	return s.settingsRepo.SetBatch(kvs)
}

// SaveInitialSettings 安装引导时写入初始设置（包含 admin）
func (s *SettingsService) SaveInitialSettings() error {
	kvs := make(map[string]string)

	siteJSON, _ := json.Marshal(s.cfg.Site)
	kvs[SettingKeySite] = string(siteJSON)

	postJSON, _ := json.Marshal(s.cfg.Post)
	kvs[SettingKeyPost] = string(postJSON)

	renderJSON, _ := json.Marshal(map[string]string{"render_mode": s.cfg.RenderMode})
	kvs[SettingKeyRender] = string(renderJSON)

	aiJSON, _ := json.Marshal(s.cfg.AI)
	kvs[SettingKeyAI] = string(aiJSON)

	adminJSON, _ := json.Marshal(s.cfg.Admin)
	kvs[SettingKeyAdmin] = string(adminJSON)

	kvs[SettingKeyDBDriver] = s.cfg.Database.Driver
	kvs[SettingKeyDBPath] = s.cfg.Database.Path

	return s.settingsRepo.SetBatch(kvs)
}

// maskAPIKey 掩盖 API Key 中间部分
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// ProfileInput 个人资料更新输入
type ProfileInput struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
	Website     string `json:"website"`
	Location    string `json:"location"`
}

// ProfileOutput 个人资料响应
type ProfileOutput struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	Avatar      string `json:"avatar"`
	Website     string `json:"website"`
	Location    string `json:"location"`
}

// GetProfile 获取当前管理员个人资料
func (s *SettingsService) GetProfile() *ProfileOutput {
	p := s.cfg.Admin.Profile
	displayName := p.DisplayName
	if displayName == "" {
		displayName = s.cfg.Admin.Username
	}
	return &ProfileOutput{
		Username:    s.cfg.Admin.Username,
		DisplayName: displayName,
		Email:       p.Email,
		Bio:         p.Bio,
		Avatar:      p.Avatar,
		Website:     p.Website,
		Location:    p.Location,
	}
}

// UpdateProfile 更新管理员个人资料并持久化
func (s *SettingsService) UpdateProfile(input ProfileInput) (*ProfileOutput, error) {
	s.cfg.Admin.Profile.DisplayName = input.DisplayName
	s.cfg.Admin.Profile.Email = input.Email
	s.cfg.Admin.Profile.Bio = input.Bio
	s.cfg.Admin.Profile.Avatar = input.Avatar
	s.cfg.Admin.Profile.Website = input.Website
	s.cfg.Admin.Profile.Location = input.Location

	if err := s.saveAdminToDB(); err != nil {
		return nil, fmt.Errorf("保存个人资料失败: %w", err)
	}
	return s.GetProfile(), nil
}

// saveAdminToDB 将 admin 配置持久化到 DB
func (s *SettingsService) saveAdminToDB() error {
	adminJSON, _ := json.Marshal(s.cfg.Admin)
	return s.settingsRepo.SetBatch(map[string]string{
		SettingKeyAdmin: string(adminJSON),
	})
}

// NavMenuItem 导航菜单项（最多支持二级）
type NavMenuItem struct {
	Title        string         `json:"title"`
	URL          string         `json:"url"`
	Icon         string         `json:"icon,omitempty"`
	OpenInNewTab bool           `json:"open_in_new_tab"`
	Children     []NavMenuItem  `json:"children,omitempty"`
}

// GetNavMenus 获取导航菜单列表
func (s *SettingsService) GetNavMenus() []NavMenuItem {
	raw := s.settingsRepo.Get(SettingKeyNavMenus)
	if raw == "" {
		return nil
	}
	var menus []NavMenuItem
	if err := json.Unmarshal([]byte(raw), &menus); err != nil {
		return nil
	}
	return menus
}

// SaveNavMenus 保存导航菜单列表
func (s *SettingsService) SaveNavMenus(menus []NavMenuItem) error {
	data, err := json.Marshal(menus)
	if err != nil {
		return fmt.Errorf("序列化菜单失败: %w", err)
	}
	return s.settingsRepo.Set(SettingKeyNavMenus, string(data))
}
