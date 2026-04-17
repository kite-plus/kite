package model

import "time"

// FileReplica 文件的副本记录。
// 仅在上传策略为 mirror（双备份/多备份）时产生；主位置仍记在 files.storage_config_id。
// storage_key 与 files.storage_key 相同，副本在不同存储后端使用同一路径。
type FileReplica struct {
	ID              string    `gorm:"column:id;primaryKey" json:"id"`
	FileID          string    `gorm:"column:file_id;index;not null" json:"file_id"`
	StorageConfigID string    `gorm:"column:storage_config_id;index;not null" json:"storage_config_id"`
	Status          string    `gorm:"column:status;not null;default:'pending'" json:"status"` // pending / ok / failed
	ErrorMsg        string    `gorm:"column:error_msg" json:"error_msg,omitempty"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (FileReplica) TableName() string { return "file_replicas" }

// 副本状态常量。
const (
	ReplicaStatusPending = "pending"
	ReplicaStatusOK      = "ok"
	ReplicaStatusFailed  = "failed"
)
