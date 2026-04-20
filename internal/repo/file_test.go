package repo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/model"
)

// makeFile is a convenience helper for building model.File test fixtures.
func makeFile(id, userID, fileType string, sizeBytes int64) *model.File {
	return &model.File{
		ID:              id,
		UserID:          userID,
		StorageConfigID: "s1",
		OriginalName:    id + ".bin",
		StorageKey:      "keys/" + id,
		HashMD5:         id + "00000000000000000000000000000000", // 32-char placeholder
		SizeBytes:       sizeBytes,
		MimeType:        "application/octet-stream",
		FileType:        fileType,
		URL:             "https://example.com/" + id,
		CreatedAt:       time.Now(),
	}
}

func TestFileRepo_CreateAndGetByID(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f := makeFile("fi1", "u1", model.FileTypeImage, 512)
	if err := r.Create(ctx, f); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.GetByID(ctx, "fi1")
	if err != nil || got.UserID != "u1" {
		t.Fatalf("GetByID: err=%v got=%v", err, got)
	}
}

func TestFileRepo_GetByID_NotFound(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	_, err := r.GetByID(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileRepo_GetByID_SoftDeleted(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f := makeFile("fi2", "u1", model.FileTypeFile, 100)
	r.Create(ctx, f)
	r.SoftDelete(ctx, "fi2")

	_, err := r.GetByID(ctx, "fi2")
	if err == nil {
		t.Fatal("soft-deleted file should not be found by GetByID")
	}
}

func TestFileRepo_GetByHashMD5(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f := makeFile("fi3", "u1", model.FileTypeImage, 200)
	r.Create(ctx, f)

	got, err := r.GetByHashMD5(ctx, "u1", f.HashMD5)
	if err != nil || got.ID != "fi3" {
		t.Fatalf("GetByHashMD5: err=%v", err)
	}

	_, err = r.GetByHashMD5(ctx, "u1", "deadbeef00000000000000000000dead")
	if err == nil {
		t.Fatal("expected error for missing MD5")
	}
}

func TestFileRepo_GetByHashPrefix(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f := makeFile("fi4", "u1", model.FileTypeImage, 300)
	r.Create(ctx, f)

	// Prefix too short.
	_, err := r.GetByHashPrefix(ctx, "abc")
	if err == nil {
		t.Fatal("expected error for prefix < 8 chars")
	}

	// Valid prefix from the MD5.
	prefix := f.HashMD5[:8]
	got, err := r.GetByHashPrefix(ctx, prefix)
	if err != nil || got.ID != "fi4" {
		t.Fatalf("GetByHashPrefix: err=%v got=%v", err, got)
	}

	// Non-matching prefix.
	_, err = r.GetByHashPrefix(ctx, "00000000")
	if err == nil {
		t.Fatal("expected error for non-matching prefix")
	}
}

func TestFileRepo_List_UserFilter(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("lf1", "u1", model.FileTypeImage, 1))
	r.Create(ctx, makeFile("lf2", "u2", model.FileTypeImage, 1))

	files, total, err := r.List(ctx, FileListParams{UserID: "u1", Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(files) != 1 {
		t.Fatalf("List UserFilter: err=%v total=%d len=%d", err, total, len(files))
	}
}

func TestFileRepo_List_TypeFilter(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("tf1", "u1", model.FileTypeImage, 1))
	r.Create(ctx, makeFile("tf2", "u1", model.FileTypeVideo, 1))

	files, total, _ := r.List(ctx, FileListParams{FileType: model.FileTypeVideo, Page: 1, PageSize: 10})
	if total != 1 || files[0].ID != "tf2" {
		t.Fatalf("List TypeFilter: total=%d", total)
	}
}

func TestFileRepo_List_KeywordFilter(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f := makeFile("kf1", "u1", model.FileTypeImage, 1)
	f.OriginalName = "vacation-photo.jpg"
	r.Create(ctx, f)

	f2 := makeFile("kf2", "u1", model.FileTypeImage, 1)
	f2.OriginalName = "work-report.pdf"
	r.Create(ctx, f2)

	files, total, _ := r.List(ctx, FileListParams{Keyword: "vacation", Page: 1, PageSize: 10})
	if total != 1 || files[0].ID != "kf1" {
		t.Fatalf("List Keyword: total=%d", total)
	}
}

func TestFileRepo_List_NoAlbumFilter(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f1 := makeFile("na1", "u1", model.FileTypeFile, 1)
	albumID := "al1"
	f1.AlbumID = &albumID

	f2 := makeFile("na2", "u1", model.FileTypeFile, 1)

	r.Create(ctx, f1)
	r.Create(ctx, f2)

	files, total, _ := r.List(ctx, FileListParams{NoAlbum: true, Page: 1, PageSize: 10})
	if total != 1 || files[0].ID != "na2" {
		t.Fatalf("List NoAlbum: total=%d", total)
	}
}

func TestFileRepo_List_AlbumFilter(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	albumID := "al99"
	f1 := makeFile("af1", "u1", model.FileTypeFile, 1)
	f1.AlbumID = &albumID
	f2 := makeFile("af2", "u1", model.FileTypeFile, 1)

	r.Create(ctx, f1)
	r.Create(ctx, f2)

	files, total, _ := r.List(ctx, FileListParams{AlbumID: "al99", Page: 1, PageSize: 10})
	if total != 1 || files[0].ID != "af1" {
		t.Fatalf("List AlbumFilter: total=%d", total)
	}
}

func TestFileRepo_List_Pagination(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		r.Create(ctx, makeFile(fmt.Sprintf("pg%d", i), "u1", model.FileTypeFile, 1))
	}

	_, total, _ := r.List(ctx, FileListParams{Page: 1, PageSize: 3})
	if total != 5 {
		t.Fatalf("expected total 5, got %d", total)
	}

	page2, _, _ := r.List(ctx, FileListParams{Page: 2, PageSize: 3})
	if len(page2) != 2 {
		t.Fatalf("page2: expected 2, got %d", len(page2))
	}
}

func TestFileRepo_SoftDelete(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("sd1", "u1", model.FileTypeFile, 1))
	if err := r.SoftDelete(ctx, "sd1"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// Row must still be in DB.
	var found model.File
	r.db.Where("id = ?", "sd1").First(&found)
	if !found.IsDeleted {
		t.Fatal("expected is_deleted=true")
	}
}

func TestFileRepo_BatchSoftDelete(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("bd1", "u1", model.FileTypeFile, 1))
	r.Create(ctx, makeFile("bd2", "u1", model.FileTypeFile, 1))

	if err := r.BatchSoftDelete(ctx, []string{"bd1", "bd2"}); err != nil {
		t.Fatalf("BatchSoftDelete: %v", err)
	}

	_, err1 := r.GetByID(ctx, "bd1")
	_, err2 := r.GetByID(ctx, "bd2")
	if err1 == nil || err2 == nil {
		t.Fatal("both files should be soft-deleted")
	}
}

func TestFileRepo_SetAlbum(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("sa1", "u1", model.FileTypeFile, 1))

	albumID := "my-album"
	if err := r.SetAlbum(ctx, "sa1", &albumID); err != nil {
		t.Fatalf("SetAlbum set: %v", err)
	}
	got, _ := r.GetByID(ctx, "sa1")
	if got.AlbumID == nil || *got.AlbumID != "my-album" {
		t.Fatalf("album_id not set: %v", got.AlbumID)
	}

	if err := r.SetAlbum(ctx, "sa1", nil); err != nil {
		t.Fatalf("SetAlbum clear: %v", err)
	}
	got, _ = r.GetByID(ctx, "sa1")
	if got.AlbumID != nil {
		t.Fatalf("album_id should be cleared: %v", got.AlbumID)
	}
}

func TestFileRepo_CountAndSumByUser(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("cs1", "u1", model.FileTypeImage, 100))
	r.Create(ctx, makeFile("cs2", "u1", model.FileTypeVideo, 200))
	r.Create(ctx, makeFile("cs3", "u2", model.FileTypeFile, 999))

	count, _ := r.CountByUser(ctx, "u1")
	if count != 2 {
		t.Fatalf("CountByUser: expected 2, got %d", count)
	}

	sum, _ := r.SumSizeByUser(ctx, "u1")
	if sum != 300 {
		t.Fatalf("SumSizeByUser: expected 300, got %d", sum)
	}

	// Soft-deleted files are excluded.
	r.SoftDelete(ctx, "cs1")
	count, _ = r.CountByUser(ctx, "u1")
	if count != 1 {
		t.Fatalf("CountByUser after delete: expected 1, got %d", count)
	}
}

func TestFileRepo_SumSizeByStorageConfig(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	f1 := makeFile("sc1", "u1", model.FileTypeFile, 400)
	f1.StorageConfigID = "cfg-A"
	f2 := makeFile("sc2", "u1", model.FileTypeFile, 600)
	f2.StorageConfigID = "cfg-B"
	r.Create(ctx, f1)
	r.Create(ctx, f2)

	sum, _ := r.SumSizeByStorageConfig(ctx, "cfg-A")
	if sum != 400 {
		t.Fatalf("SumSizeByStorageConfig: expected 400, got %d", sum)
	}
}

func TestFileRepo_CountByAlbum(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	aid := "alb-test"
	f1 := makeFile("ca1", "u1", model.FileTypeFile, 1)
	f1.AlbumID = &aid
	f2 := makeFile("ca2", "u1", model.FileTypeFile, 1)
	r.Create(ctx, f1)
	r.Create(ctx, f2)

	n, _ := r.CountByAlbum(ctx, "alb-test")
	if n != 1 {
		t.Fatalf("CountByAlbum: expected 1, got %d", n)
	}
}

func TestFileRepo_GetStats(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("gs1", "u1", model.FileTypeImage, 100))
	r.Create(ctx, makeFile("gs2", "u1", model.FileTypeVideo, 200))
	r.Create(ctx, makeFile("gs3", "u1", model.FileTypeAudio, 300))
	r.Create(ctx, makeFile("gs4", "u1", model.FileTypeFile, 400))

	stats, err := r.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalFiles != 4 {
		t.Fatalf("TotalFiles: expected 4, got %d", stats.TotalFiles)
	}
	if stats.TotalSize != 1000 {
		t.Fatalf("TotalSize: expected 1000, got %d", stats.TotalSize)
	}
	if stats.ImageCount != 1 || stats.VideoCount != 1 || stats.AudioCount != 1 || stats.OtherCount != 1 {
		t.Fatalf("per-type counts wrong: %+v", stats)
	}
}

func TestFileRepo_GetUserStats(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, makeFile("us1", "u1", model.FileTypeImage, 50))
	r.Create(ctx, makeFile("us2", "u2", model.FileTypeImage, 999))

	stats, err := r.GetUserStats(ctx, "u1")
	if err != nil {
		t.Fatalf("GetUserStats: %v", err)
	}
	if stats.TotalFiles != 1 || stats.TotalSize != 50 {
		t.Fatalf("GetUserStats: %+v", stats)
	}
}

func TestFileRepo_GetDailyUploadStats(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	now := time.Now().UTC()
	start := now.Truncate(24 * time.Hour)
	end := start.Add(48 * time.Hour)

	f := makeFile("du1", "u1", model.FileTypeFile, 1)
	f.CreatedAt = start.Add(time.Hour)
	r.Create(ctx, f)

	// File outside the window.
	f2 := makeFile("du2", "u1", model.FileTypeFile, 1)
	f2.CreatedAt = start.Add(-time.Hour)
	r.Create(ctx, f2)

	rows, err := r.GetDailyUploadStats(ctx, "u1", start, end)
	if err != nil {
		t.Fatalf("GetDailyUploadStats: %v", err)
	}
	if len(rows) != 1 || rows[0].UploadCount != 1 {
		t.Fatalf("expected 1 row with count 1, got %d rows: %v", len(rows), rows)
	}
}

func TestFileRepo_GetHourlyUploadHeatmapStats(t *testing.T) {
	r := NewFileRepo(newTestDB(t))
	ctx := context.Background()

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)

	f := makeFile("hm1", "u1", model.FileTypeFile, 1)
	f.CreatedAt = time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC) // Thursday hour 14
	r.Create(ctx, f)

	rows, err := r.GetHourlyUploadHeatmapStats(ctx, "u1", start, end)
	if err != nil {
		t.Fatalf("GetHourlyUploadHeatmapStats: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(rows))
	}
	if rows[0].Hour != 14 {
		t.Fatalf("expected hour 14, got %d", rows[0].Hour)
	}
}
