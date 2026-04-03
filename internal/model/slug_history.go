package model

import "github.com/google/uuid"

// SlugHistory 记录文章 slug 变更历史，用于旧链接 301 重定向
type SlugHistory struct {
	BaseModel
	PostID  uuid.UUID `gorm:"type:char(36);not null;index" json:"post_id"`
	OldSlug string    `gorm:"size:255;not null;uniqueIndex" json:"old_slug"`
}
