package model

import "github.com/google/uuid"

const (
	CommentStatusPending  = "pending"
	CommentStatusApproved = "approved"
	CommentStatusSpam     = "spam"
)

// Comment 评论模型
type Comment struct {
	BaseModel
	PostID    uuid.UUID  `gorm:"type:char(36);not null;index" json:"post_id"`
	Post      *Post      `json:"post,omitempty"`
	ParentID  *uuid.UUID `gorm:"type:char(36);index" json:"parent_id"`
	Author    string     `gorm:"size:255;not null" json:"author"`
	Email     string     `gorm:"size:255;not null" json:"email"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	Status    string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	IP        string     `gorm:"size:64" json:"ip"`
	UserAgent string     `gorm:"size:512" json:"user_agent"`
	Replies   []Comment  `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}
