package model

import "time"

// FileAccessLog 记录每次文件访问/下载，用于带宽与访问量统计。
// UserID 存储文件所属用户（非访问者），用于按文件拥有者维度做统计隔离；游客文件为空。
type FileAccessLog struct {
	ID          string    `gorm:"column:id;primaryKey" json:"id"`
	FileID      string    `gorm:"column:file_id;index;not null" json:"file_id"`
	UserID      string    `gorm:"column:user_id;index" json:"user_id"`
	BytesServed int64     `gorm:"column:bytes_served;not null" json:"bytes_served"`
	AccessedAt  time.Time `gorm:"column:accessed_at;autoCreateTime;index" json:"accessed_at"`
}

func (FileAccessLog) TableName() string { return "file_access_logs" }
