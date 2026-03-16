package model

import "time"

const (
	PageStatusDraft     = "draft"
	PageStatusPublished = "published"
)

// Page 独立页面模型
type Page struct {
	BaseModel
	Title           string     `gorm:"size:255;not null" json:"title"`
	Slug            string     `gorm:"size:255;not null;uniqueIndex" json:"slug"`
	ContentMarkdown string     `gorm:"type:text;not null" json:"content_markdown"`
	ContentHTML     string     `gorm:"type:text;not null" json:"content_html"`
	Status          string     `gorm:"size:32;not null;default:draft;index" json:"status"`
	SortOrder       int        `gorm:"not null;default:0;index" json:"sort_order"`
	ShowInNav       bool       `gorm:"not null;default:false" json:"show_in_nav"`
	PublishedAt     *time.Time `json:"published_at"`
	Template        string     `gorm:"size:64;not null;default:default" json:"template"` // 模板名，对应 templates/pages/{template}.html
	Config          string     `gorm:"type:text" json:"config"`                          // 模板参数，JSON 格式
}
