package model

import "github.com/google/uuid"

type Category struct {
	BaseModel
	Name        string     `gorm:"size:255;not null" json:"name"`
	Slug        string     `gorm:"size:255;not null;uniqueIndex" json:"slug"`
	Description string     `gorm:"type:text" json:"description"`
	Icon        string     `gorm:"size:100" json:"icon"`
	ParentID    *uuid.UUID `gorm:"type:char(36);index" json:"parent_id"`
	// PostCount 不存数据库，通过 SQL 聚合查询填充
	PostCount int64      `gorm:"-" json:"post_count"`
	// Children 不存数据库，由前端或服务层构建树时填充
	Children  []Category `gorm:"-" json:"children,omitempty"`
}
