package model

import "time"

const (
	PostStatusDraft     = "draft"
	PostStatusPublished = "published"
	PostStatusArchived  = "archived"
)

type Post struct {
	BaseModel
	Title       string     `gorm:"size:255;not null" json:"title"`
	Slug        string     `gorm:"size:255;not null;uniqueIndex" json:"slug"`
	Summary     string     `gorm:"type:text" json:"summary"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Status      string     `gorm:"size:32;not null;index" json:"status"`
	CoverImage  string     `gorm:"size:1024" json:"cover_image"`
	PublishedAt *time.Time `json:"published_at"`
}
