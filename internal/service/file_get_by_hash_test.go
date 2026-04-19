package service

import (
	"context"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestFileService_GetFileByHash(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.File{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	f := &model.File{
		ID:              "f-1",
		UserID:          "u-1",
		StorageConfigID: "s-1",
		OriginalName:    "demo.jpg",
		StorageKey:      "2026/04/demo.jpg",
		HashMD5:         "abcd1234abcd1234abcd1234abcd1234",
		SizeBytes:       100,
		MimeType:        "image/jpeg",
		FileType:        model.FileTypeImage,
		URL:             "/i/abcd1234",
		CreatedAt:       time.Now(),
	}
	if err := db.Create(f).Error; err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	svc := &FileService{fileRepo: repo.NewFileRepo(db)}

	got, err := svc.GetFileByHash(context.Background(), "abcd1234")
	if err != nil {
		t.Fatalf("GetFileByHash returned error: %v", err)
	}
	if got == nil || got.ID != f.ID {
		t.Fatalf("unexpected file result: %#v", got)
	}

	_, err = svc.GetFileByHash(context.Background(), "short")
	if err == nil {
		t.Fatal("expected error for short hash prefix")
	}
	if err != ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}
