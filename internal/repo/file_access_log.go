package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

// FileAccessLogRepo 文件访问日志的数据访问层。
type FileAccessLogRepo struct {
	db *gorm.DB
}

func NewFileAccessLogRepo(db *gorm.DB) *FileAccessLogRepo {
	return &FileAccessLogRepo{db: db}
}

// Create 记录一次文件访问。
func (r *FileAccessLogRepo) Create(ctx context.Context, log *model.FileAccessLog) error {
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("create file access log: %w", err)
	}
	return nil
}

// DailyAccessStat 每日访问量/带宽统计。
type DailyAccessStat struct {
	Day         string `json:"day"`          // YYYY-MM-DD
	AccessCount int64  `json:"access_count"` // 当日访问次数
	BytesServed int64  `json:"bytes_served"` // 当日输出字节数
}

// GetDailyAccessStats 获取指定时间范围内每日访问量和带宽。
// start/end 为 UTC 日期边界（start 含、end 不含）。
// userID 为空时返回全站聚合（管理员视角）；非空时仅统计该用户所属文件。
func (r *FileAccessLogRepo) GetDailyAccessStats(ctx context.Context, userID string, start, end time.Time) ([]DailyAccessStat, error) {
	var rows []DailyAccessStat
	db := r.db.WithContext(ctx).
		Model(&model.FileAccessLog{}).
		Select("strftime('%Y-%m-%d', accessed_at) as day, COUNT(*) as access_count, COALESCE(SUM(bytes_served), 0) as bytes_served").
		Where("accessed_at >= ? AND accessed_at < ?", start, end)
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if err := db.Group("day").Order("day ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get daily access stats: %w", err)
	}
	return rows, nil
}
