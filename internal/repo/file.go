package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

// allowedOrderByFields 文件列表允许的排序字段白名单，防止 SQL 注入。
var allowedOrderByFields = map[string]bool{
	"created_at":    true,
	"updated_at":    true,
	"size_bytes":    true,
	"original_name": true,
	"file_type":     true,
}

func sanitizeOrderBy(v string) string {
	if allowedOrderByFields[v] {
		return v
	}
	return "created_at"
}

func sanitizeOrder(v string) string {
	switch strings.ToUpper(v) {
	case "ASC":
		return "ASC"
	case "DESC":
		return "DESC"
	default:
		return "DESC"
	}
}

// FileRepo 文件数据访问层。
type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

// Create 创建文件记录。
func (r *FileRepo) Create(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	return nil
}

// GetByID 通过 ID 查询文件。
func (r *FileRepo) GetByID(ctx context.Context, id string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&file).Error; err != nil {
		return nil, fmt.Errorf("get file by id: %w", err)
	}
	return &file, nil
}

// GetByHashMD5 通过 MD5 哈希查询文件（用于去重）。
func (r *FileRepo) GetByHashMD5(ctx context.Context, userID, hashMD5 string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND hash_md5 = ? AND is_deleted = ?", userID, hashMD5, false).
		First(&file).Error; err != nil {
		return nil, fmt.Errorf("get file by md5: %w", err)
	}
	return &file, nil
}

// minHashPrefixLen 短链 hash 前缀的最小长度。
// 8 hex 字符 = 32 bit，足够避免枚举攻击，同时与 path_pattern 中的 {md5_8} 对齐。
const minHashPrefixLen = 8

// GetByHashPrefix 通过 MD5 哈希前缀查询文件，用于短链访问。
// 要求前缀至少 minHashPrefixLen 个字符，且结果按 id 排序保证确定性。
func (r *FileRepo) GetByHashPrefix(ctx context.Context, prefix string) (*model.File, error) {
	if len(prefix) < minHashPrefixLen {
		return nil, fmt.Errorf("hash prefix too short")
	}
	var file model.File
	if err := r.db.WithContext(ctx).
		Where("hash_md5 LIKE ? AND is_deleted = ?", prefix+"%", false).
		Order("id ASC").
		First(&file).Error; err != nil {
		return nil, fmt.Errorf("get file by hash prefix: %w", err)
	}
	return &file, nil
}

// FileListParams 文件列表查询参数。
type FileListParams struct {
	UserID   string
	AlbumID  string // 为空则不按相册过滤
	NoAlbum  bool   // 为 true 时查询未归属任何文件夹的文件（album_id IS NULL）
	FileType string // 为空则不按类型过滤
	Keyword  string // 搜索原始文件名
	Page     int
	PageSize int
	OrderBy  string // 排序字段，默认 created_at
	Order    string // ASC / DESC，默认 DESC
}

// List 分页查询文件列表。
func (r *FileRepo) List(ctx context.Context, params FileListParams) ([]model.File, int64, error) {
	files := make([]model.File, 0)
	var total int64

	db := r.db.WithContext(ctx).Model(&model.File{}).Where("is_deleted = ?", false)

	if params.UserID != "" {
		db = db.Where("user_id = ?", params.UserID)
	}
	if params.NoAlbum {
		db = db.Where("album_id IS NULL")
	} else if params.AlbumID != "" {
		db = db.Where("album_id = ?", params.AlbumID)
	}
	if params.FileType != "" {
		db = db.Where("file_type = ?", params.FileType)
	}
	if params.Keyword != "" {
		db = db.Where("original_name LIKE ?", "%"+params.Keyword+"%")
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count files: %w", err)
	}

	orderBy := sanitizeOrderBy(params.OrderBy)
	order := sanitizeOrder(params.Order)

	offset := (params.Page - 1) * params.PageSize
	if err := db.Order(orderBy + " " + order).
		Offset(offset).Limit(params.PageSize).
		Find(&files).Error; err != nil {
		return nil, 0, fmt.Errorf("list files: %w", err)
	}

	return files, total, nil
}

// SoftDelete 软删除文件。
func (r *FileRepo) SoftDelete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error; err != nil {
		return fmt.Errorf("soft delete file: %w", err)
	}
	return nil
}

// SetAlbum 设置或清除文件的文件夹（album_id）。albumID 为 nil 时清除归属。
func (r *FileRepo) SetAlbum(ctx context.Context, fileID string, albumID *string) error {
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("id = ? AND is_deleted = ?", fileID, false).
		Update("album_id", albumID).Error; err != nil {
		return fmt.Errorf("set album: %w", err)
	}
	return nil
}

// BatchSoftDelete 批量软删除文件。
func (r *FileRepo) BatchSoftDelete(ctx context.Context, ids []string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id IN ?", ids).
		Update("is_deleted", true).Error; err != nil {
		return fmt.Errorf("batch soft delete files: %w", err)
	}
	return nil
}

// CountByUser 统计用户的文件数量。
func (r *FileRepo) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count files by user: %w", err)
	}
	return count, nil
}

// SumSizeByUser 统计用户的文件总大小。
func (r *FileRepo) SumSizeByUser(ctx context.Context, userID string) (int64, error) {
	var total *int64
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Select("COALESCE(SUM(size_bytes), 0)").
		Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("sum file size by user: %w", err)
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// SumSizeByStorageConfig 统计指定存储配置下所有未删除文件的总大小。
func (r *FileRepo) SumSizeByStorageConfig(ctx context.Context, storageConfigID string) (int64, error) {
	var total *int64
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("storage_config_id = ? AND is_deleted = ?", storageConfigID, false).
		Select("COALESCE(SUM(size_bytes), 0)").
		Scan(&total).Error; err != nil {
		return 0, fmt.Errorf("sum file size by storage: %w", err)
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// CountByAlbum 统计相册中的文件数量。
func (r *FileRepo) CountByAlbum(ctx context.Context, albumID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("album_id = ? AND is_deleted = ?", albumID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count files by album: %w", err)
	}
	return count, nil
}

// UpdateAlbumID 修改文件所属相册。
func (r *FileRepo) UpdateAlbumID(ctx context.Context, fileID string, albumID *string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ?", fileID).
		Update("album_id", albumID).Error; err != nil {
		return fmt.Errorf("update file album: %w", err)
	}
	return nil
}

// Stats 全站文件统计。
type FileStats struct {
	TotalFiles int64 `json:"total_files"`
	TotalSize  int64 `json:"total_size"`
	ImageCount int64 `json:"image_count"`
	VideoCount int64 `json:"video_count"`
	AudioCount int64 `json:"audio_count"`
	OtherCount int64 `json:"other_count"`
	ImageSize  int64 `json:"image_size"`
	VideoSize  int64 `json:"video_size"`
	AudioSize  int64 `json:"audio_size"`
	OtherSize  int64 `json:"other_size"`
}

// GetStats 获取全站文件统计数据。
func (r *FileRepo) GetStats(ctx context.Context) (*FileStats, error) {
	var stats FileStats
	db := r.db.WithContext(ctx).Model(&model.File{}).Where("is_deleted = ?", false)

	if err := db.Count(&stats.TotalFiles).Error; err != nil {
		return nil, fmt.Errorf("count total files: %w", err)
	}

	if err := db.Select("COALESCE(SUM(size_bytes), 0)").Scan(&stats.TotalSize).Error; err != nil {
		return nil, fmt.Errorf("sum total size: %w", err)
	}

	for _, ft := range []struct {
		typ   string
		count *int64
		size  *int64
	}{
		{"image", &stats.ImageCount, &stats.ImageSize},
		{"video", &stats.VideoCount, &stats.VideoSize},
		{"audio", &stats.AudioCount, &stats.AudioSize},
		{"file", &stats.OtherCount, &stats.OtherSize},
	} {
	} {
		if err := r.db.WithContext(ctx).Model(&model.File{}).
			Where("is_deleted = ? AND file_type = ?", false, ft.typ).
			Count(ft.count).Error; err != nil {
			return nil, fmt.Errorf("count %s files: %w", ft.typ, err)
		if err := r.db.WithContext(ctx).Model(&model.File{}).
			Where("is_deleted = ? AND file_type = ?", false, ft.typ).
			Select("COALESCE(SUM(size_bytes), 0)").Scan(ft.size).Error; err != nil {
			return nil, fmt.Errorf("sum %s size: %w", ft.typ, err)
		}
		}
	}

	return &stats, nil
}

// GetUserStats 获取指定用户的文件统计。
func (r *FileRepo) GetUserStats(ctx context.Context, userID string) (*FileStats, error) {
	var stats FileStats
	db := r.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ? AND is_deleted = ?", userID, false)

	if err := db.Count(&stats.TotalFiles).Error; err != nil {
		return nil, fmt.Errorf("count user files: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Select("COALESCE(SUM(size_bytes), 0)").Scan(&stats.TotalSize).Error; err != nil {
		return nil, fmt.Errorf("sum user file size: %w", err)
	}

	for _, ft := range []struct {
		typ   string
		count *int64
		size  *int64
	}{
		{"image", &stats.ImageCount, &stats.ImageSize},
		{"video", &stats.VideoCount, &stats.VideoSize},
		{"audio", &stats.AudioCount, &stats.AudioSize},
		{"file", &stats.OtherCount, &stats.OtherSize},
	} {
		if err := r.db.WithContext(ctx).Model(&model.File{}).
			Where("user_id = ? AND is_deleted = ? AND file_type = ?", userID, false, ft.typ).
			Count(ft.count).Error; err != nil {
			return nil, fmt.Errorf("count user %s files: %w", ft.typ, err)
		}
		if err := r.db.WithContext(ctx).Model(&model.File{}).
			Where("user_id = ? AND is_deleted = ? AND file_type = ?", userID, false, ft.typ).
			Select("COALESCE(SUM(size_bytes), 0)").
			Scan(ft.size).Error; err != nil {
			return nil, fmt.Errorf("sum user %s size: %w", ft.typ, err)
		}
	}

	return &stats, nil
}

// DailyUploadStat 每日上传量统计。
type DailyUploadStat struct {
	Day         string `json:"day"`          // YYYY-MM-DD
	UploadCount int64  `json:"upload_count"` // 当日新增文件数
}

// HourlyWeekdayStat 按周几与小时聚合的统计。
// Weekday: 0=Sunday ... 6=Saturday（SQLite strftime('%w') 约定）
// Hour: 0..23
type HourlyWeekdayStat struct {
	Weekday int   `json:"weekday"`
	Hour    int   `json:"hour"`
	Count   int64 `json:"count"`
}

// GetDailyUploadStats 获取指定时间范围内每日上传量。
// start/end 为 UTC 日期边界（start 含、end 不含）。
// userID 为空时返回全站聚合（管理员视角）；非空时仅统计该用户的上传。
func (r *FileRepo) GetDailyUploadStats(ctx context.Context, userID string, start, end time.Time) ([]DailyUploadStat, error) {
	var rows []DailyUploadStat
	db := r.db.WithContext(ctx).
		Model(&model.File{}).
		Select("strftime('%Y-%m-%d', created_at) as day, COUNT(*) as upload_count").
		Where("is_deleted = ? AND created_at >= ? AND created_at < ?", false, start, end)
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if err := db.Group("day").Order("day ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get daily upload stats: %w", err)
	}
	return rows, nil
}

// GetHourlyUploadHeatmapStats 获取指定时间范围内按周几与小时聚合的上传热力图统计。
// start/end 为时间边界（start 含、end 不含）。
// userID 为空时返回全站聚合（管理员视角）；非空时仅统计该用户上传。
func (r *FileRepo) GetHourlyUploadHeatmapStats(ctx context.Context, userID string, start, end time.Time) ([]HourlyWeekdayStat, error) {
	var rows []HourlyWeekdayStat
	db := r.db.WithContext(ctx).
		Model(&model.File{}).
		Select("CAST(strftime('%w', created_at) AS INTEGER) as weekday, CAST(strftime('%H', created_at) AS INTEGER) as hour, COUNT(*) as count").
		Where("is_deleted = ? AND created_at >= ? AND created_at < ?", false, start, end)
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if err := db.Group("weekday, hour").Order("weekday ASC, hour ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get hourly upload heatmap stats: %w", err)
	}
	return rows, nil
}
