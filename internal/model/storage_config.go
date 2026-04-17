package model

import "time"

// StorageConfig 存储配置模型。
// 系统支持多个存储配置，管理员可以配置本地、S3 兼容、阿里云 OSS、腾讯云 COS 或 FTP 存储。
// config 字段为 JSON 序列化的配置详情。
type StorageConfig struct {
	ID                 string    `gorm:"column:id;primaryKey" json:"id"`
	Name               string    `gorm:"column:name;not null" json:"name"`
	Driver             string    `gorm:"column:driver;not null" json:"driver"` // local / s3 / oss / cos / ftp
	Config             string    `gorm:"column:config;not null" json:"-"`
	CapacityLimitBytes int64     `gorm:"column:capacity_limit_bytes;default:0" json:"capacity_limit_bytes"` // 容量上限（字节），0 表示不限制
	Priority           int       `gorm:"column:priority;default:100;index" json:"priority"`                 // 优先级，越小越优先，用于 fallback / round_robin / mirror
	IsDefault          bool      `gorm:"column:is_default;default:false" json:"is_default"`
	IsActive           bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (StorageConfig) TableName() string { return "storage_configs" }
