package model

// Setting 系统设置（key-value 存储）
type Setting struct {
	Key   string `gorm:"primaryKey;size:100" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}
