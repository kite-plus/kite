package model

type Tag struct {
	BaseModel
	Name string `gorm:"size:255;not null" json:"name"`
	Slug string `gorm:"size:255;not null;uniqueIndex" json:"slug"`
}
