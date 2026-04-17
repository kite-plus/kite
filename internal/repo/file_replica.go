package repo

import (
	"context"
	"fmt"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

// FileReplicaRepo 文件副本（mirror 模式）数据访问层。
type FileReplicaRepo struct {
	db *gorm.DB
}

func NewFileReplicaRepo(db *gorm.DB) *FileReplicaRepo {
	return &FileReplicaRepo{db: db}
}

// Create 新增副本记录，一般在上传流程中以 pending 状态预写入。
func (r *FileReplicaRepo) Create(ctx context.Context, replica *model.FileReplica) error {
	if err := r.db.WithContext(ctx).Create(replica).Error; err != nil {
		return fmt.Errorf("create file replica: %w", err)
	}
	return nil
}

// ListByFile 返回指定文件的所有副本记录。
func (r *FileReplicaRepo) ListByFile(ctx context.Context, fileID string) ([]model.FileReplica, error) {
	var replicas []model.FileReplica
	if err := r.db.WithContext(ctx).
		Where("file_id = ?", fileID).
		Find(&replicas).Error; err != nil {
		return nil, fmt.Errorf("list file replicas: %w", err)
	}
	return replicas, nil
}

// UpdateStatus 更新副本写入结果。
func (r *FileReplicaRepo) UpdateStatus(ctx context.Context, id, status, errMsg string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.FileReplica{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":    status,
			"error_msg": errMsg,
		}).Error; err != nil {
		return fmt.Errorf("update file replica status: %w", err)
	}
	return nil
}

// DeleteByFile 删除指定文件的所有副本记录。
func (r *FileReplicaRepo) DeleteByFile(ctx context.Context, fileID string) error {
	if err := r.db.WithContext(ctx).
		Where("file_id = ?", fileID).
		Delete(&model.FileReplica{}).Error; err != nil {
		return fmt.Errorf("delete file replicas: %w", err)
	}
	return nil
}
