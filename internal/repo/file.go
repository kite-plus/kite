package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kite-plus/kite/internal/model"
	"gorm.io/gorm"
)

// allowedOrderByFields is the whitelist of sort columns for the file list, used to prevent SQL injection.
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

// FileRepo is the data access layer for files.
type FileRepo struct {
	db *gorm.DB
}

func NewFileRepo(db *gorm.DB) *FileRepo {
	return &FileRepo{db: db}
}

// Create inserts a new file record.
func (r *FileRepo) Create(ctx context.Context, file *model.File) error {
	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	return nil
}

// GetByID fetches a file by its ID.
func (r *FileRepo) GetByID(ctx context.Context, id string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&file).Error; err != nil {
		return nil, fmt.Errorf("get file by id: %w", err)
	}
	return &file, nil
}

// GetByHashMD5 fetches a file by its MD5 hash, used for deduplication.
func (r *FileRepo) GetByHashMD5(ctx context.Context, userID, hashMD5 string) (*model.File, error) {
	var file model.File
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND hash_md5 = ? AND is_deleted = ?", userID, hashMD5, false).
		First(&file).Error; err != nil {
		return nil, fmt.Errorf("get file by md5: %w", err)
	}
	return &file, nil
}

// ListByHashMD5 fetches all files matching the same owner and MD5 hash.
func (r *FileRepo) ListByHashMD5(ctx context.Context, userID, hashMD5 string) ([]model.File, error) {
	files := make([]model.File, 0)
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND hash_md5 = ? AND is_deleted = ?", userID, hashMD5, false).
		Order("created_at ASC, id ASC").
		Find(&files).Error; err != nil {
		return nil, fmt.Errorf("list files by md5: %w", err)
	}
	return files, nil
}

// minHashPrefixLen is the minimum length of the short-link hash prefix.
// 8 hex characters (32 bits) is enough to avoid enumeration and matches {md5_8} in path_pattern.
const minHashPrefixLen = 8

// GetByHashPrefix fetches a file by its MD5 hash prefix for short-link access.
// The prefix must be at least minHashPrefixLen characters, and results are ordered by id for determinism.
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

// FileListParams holds the query parameters for listing files.
type FileListParams struct {
	UserID   string
	AlbumID  string // empty skips the album filter
	NoAlbum  bool   // true selects files not in any folder (album_id IS NULL)
	FileType string // empty skips the type filter
	Keyword  string // substring match against the original filename
	Page     int
	PageSize int
	OrderBy  string // sort column; defaults to created_at
	Order    string // ASC / DESC; defaults to DESC
}

// List returns a paginated slice of files.
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

// SoftDelete marks a file as deleted without removing the row.
func (r *FileRepo) SoftDelete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error; err != nil {
		return fmt.Errorf("soft delete file: %w", err)
	}
	return nil
}

// SetAlbum assigns or clears the folder of a file. A nil albumID clears the association.
func (r *FileRepo) SetAlbum(ctx context.Context, fileID string, albumID *string) error {
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("id = ? AND is_deleted = ?", fileID, false).
		Update("album_id", albumID).Error; err != nil {
		return fmt.Errorf("set album: %w", err)
	}
	return nil
}

// UpdateThumbURL sets the file's thumbnail short-link. Pass nil to clear it.
func (r *FileRepo) UpdateThumbURL(ctx context.Context, fileID string, thumbURL *string) error {
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("id = ?", fileID).
		Update("thumb_url", thumbURL).Error; err != nil {
		return fmt.Errorf("update thumb url: %w", err)
	}
	return nil
}

// BatchSoftDelete soft-deletes multiple files at once.
func (r *FileRepo) BatchSoftDelete(ctx context.Context, ids []string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id IN ?", ids).
		Update("is_deleted", true).Error; err != nil {
		return fmt.Errorf("batch soft delete files: %w", err)
	}
	return nil
}

// CountByUser returns the number of files owned by a user.
func (r *FileRepo) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count files by user: %w", err)
	}
	return count, nil
}

// SumSizeByUser returns the total byte size of a user's files.
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

// SumSizeByStorageConfig returns the total byte size of non-deleted files stored under the given storage config.
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

// CountByAlbum returns the number of files in an album.
func (r *FileRepo) CountByAlbum(ctx context.Context, albumID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("album_id = ? AND is_deleted = ?", albumID, false).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count files by album: %w", err)
	}
	return count, nil
}

// UpdateAlbumID reassigns a file to a different album.
func (r *FileRepo) UpdateAlbumID(ctx context.Context, fileID string, albumID *string) error {
	if err := r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ?", fileID).
		Update("album_id", albumID).Error; err != nil {
		return fmt.Errorf("update file album: %w", err)
	}
	return nil
}

// FileStats holds site-wide or per-user file statistics.
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

// fileTypeAggRow is the per-file-type aggregate scanned out of the
// stats GROUP BY query. The column tags match the SELECT aliases below
// so GORM can hydrate the struct directly without an intermediate map.
type fileTypeAggRow struct {
	FileType  string `gorm:"column:file_type"`
	Cnt       int64  `gorm:"column:cnt"`
	TotalSize int64  `gorm:"column:total_size"`
}

// applyFileTypeAggregate folds a slice of GROUP BY rows into a FileStats
// value. TotalFiles / TotalSize accumulate every row (so unknown
// file_type values still count toward the totals, matching the original
// COUNT(*) behaviour); the per-type buckets only react to the four
// known categories defined in [model.FileType*].
func applyFileTypeAggregate(stats *FileStats, rows []fileTypeAggRow) {
	for _, row := range rows {
		stats.TotalFiles += row.Cnt
		stats.TotalSize += row.TotalSize
		switch row.FileType {
		case model.FileTypeImage:
			stats.ImageCount = row.Cnt
			stats.ImageSize = row.TotalSize
		case model.FileTypeVideo:
			stats.VideoCount = row.Cnt
			stats.VideoSize = row.TotalSize
		case model.FileTypeAudio:
			stats.AudioCount = row.Cnt
			stats.AudioSize = row.TotalSize
		case model.FileTypeFile:
			stats.OtherCount = row.Cnt
			stats.OtherSize = row.TotalSize
		}
	}
}

// GetStats returns site-wide file statistics.
//
// A single GROUP BY file_type query covers both the count and the byte
// total for every category in one round-trip. Earlier revisions ran
// ten queries here (two for the totals plus eight for the four typed
// buckets), which was wasteful at any scale and visibly slow on
// instances backed by a remote MySQL/Postgres. Folding the totals from
// the GROUP BY rows preserves the original COUNT(*) semantics — rows
// with an unknown file_type still feed TotalFiles/TotalSize, they just
// don't land in any per-type bucket.
func (r *FileRepo) GetStats(ctx context.Context) (*FileStats, error) {
	var rows []fileTypeAggRow
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("is_deleted = ?", false).
		Select("file_type, COUNT(*) AS cnt, COALESCE(SUM(size_bytes), 0) AS total_size").
		Group("file_type").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("aggregate file stats: %w", err)
	}

	var stats FileStats
	applyFileTypeAggregate(&stats, rows)
	return &stats, nil
}

// GetUserStats returns file statistics scoped to a single user.
//
// Mirrors GetStats — one GROUP BY query, one allocation for the
// aggregate slice — with the user_id predicate added. See GetStats for
// the reasoning behind the GROUP BY collapse.
func (r *FileRepo) GetUserStats(ctx context.Context, userID string) (*FileStats, error) {
	var rows []fileTypeAggRow
	if err := r.db.WithContext(ctx).Model(&model.File{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Select("file_type, COUNT(*) AS cnt, COALESCE(SUM(size_bytes), 0) AS total_size").
		Group("file_type").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("aggregate user file stats: %w", err)
	}

	var stats FileStats
	applyFileTypeAggregate(&stats, rows)
	return &stats, nil
}

// DailyUploadStat captures the per-day upload count.
type DailyUploadStat struct {
	Day         string `json:"day"`          // YYYY-MM-DD
	UploadCount int64  `json:"upload_count"` // files added that day
}

// HourlyWeekdayStat aggregates counts by weekday and hour.
// Weekday: 0=Sunday ... 6=Saturday (SQLite strftime('%w') convention).
// Hour: 0..23.
type HourlyWeekdayStat struct {
	Weekday int   `json:"weekday"`
	Hour    int   `json:"hour"`
	Count   int64 `json:"count"`
}

// GetDailyUploadStats returns per-day upload counts in [start, end).
// start/end are UTC day boundaries (start inclusive, end exclusive).
// An empty userID returns site-wide aggregates (admin view); otherwise results are scoped to that user's uploads.
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

// GetHourlyUploadHeatmapStats returns upload counts grouped by weekday and hour within [start, end).
// start/end are time boundaries (start inclusive, end exclusive).
// An empty userID returns site-wide aggregates (admin view); otherwise results are scoped to that user's uploads.
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
