package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	PostStatusDraft     = "draft"
	PostStatusPublished = "published"
	PostStatusArchived  = "archived"
)

type Post struct {
	BaseModel
	Title           string     `gorm:"size:255;not null" json:"title"`
	Slug            string     `gorm:"size:255;not null;uniqueIndex" json:"slug"`
	Summary         string     `gorm:"type:text" json:"summary"`
	ContentMarkdown string     `gorm:"type:text;not null" json:"content_markdown"`
	ContentHTML     string     `gorm:"type:text;not null" json:"content_html"`
	Status          string     `gorm:"size:32;not null;index" json:"status"`
	CoverImage      string     `gorm:"size:1024" json:"cover_image"`
	Password        string     `gorm:"size:255" json:"-"`
	PublishedAt     *time.Time `json:"published_at"`
	ShowComments    bool       `gorm:"not null;default:true" json:"show_comments"`
	CategoryID      *uuid.UUID `gorm:"type:char(36);index" json:"category_id"`
	Category        *Category  `json:"category,omitempty"`
	Tags            []Tag      `gorm:"many2many:post_tags;" json:"tags,omitempty"`
}
