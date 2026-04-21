package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/model"
)

func TestFileService_UploadBlocksDangerousExtensionByDefault(t *testing.T) {
	svc, cleanup, _ := newUploadPathTestService(t)
	defer cleanup()

	_, err := svc.Upload(context.Background(), UploadParams{
		UserID:   "u1",
		Filename: "setup.exe",
		Reader:   bytes.NewReader([]byte("hello")),
		Size:     int64(len([]byte("hello"))),
		BaseURL:  "http://localhost:8080",
	})
	if !errors.Is(err, ErrFileTypeDenied) {
		t.Fatalf("expected ErrFileTypeDenied, got %v", err)
	}
}

func TestFileService_UploadRenamesDangerousExtension(t *testing.T) {
	svc, cleanup, settingRepo := newUploadPathTestService(t)
	defer cleanup()

	if err := settingRepo.Set(context.Background(), UploadDangerousExtensionRulesSettingKey, `[{"ext":".exe","action":"rename"}]`); err != nil {
		t.Fatalf("Set upload.dangerous_extension_rules: %v", err)
	}

	body := []byte("renamed dangerous upload")
	result, err := svc.Upload(context.Background(), UploadParams{
		UserID:   "u1",
		Filename: "setup.exe",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if result.File.OriginalName != "setup.exe.blocked" {
		t.Fatalf("unexpected original name: %q", result.File.OriginalName)
	}
	if result.File.MimeType != DangerousRenameMimeType {
		t.Fatalf("unexpected mime type: %q", result.File.MimeType)
	}
	if result.File.FileType != model.FileTypeFile {
		t.Fatalf("unexpected file type: %q", result.File.FileType)
	}
	if result.File.ThumbURL != nil {
		t.Fatal("renamed dangerous file should not have thumbnail")
	}
	if !strings.HasPrefix(result.File.URL, "/f/") {
		t.Fatalf("unexpected access url: %q", result.File.URL)
	}
	if !strings.HasSuffix(result.File.StorageKey, ".blocked") {
		t.Fatalf("unexpected storage key: %q", result.File.StorageKey)
	}
}

func TestFileService_UploadRenamedDangerousExtensionSkipsUnsafeDuplicate(t *testing.T) {
	svc, cleanup, settingRepo := newUploadPathTestService(t)
	defer cleanup()

	if err := settingRepo.Set(context.Background(), UploadDangerousExtensionRulesSettingKey, `[{"ext":".exe","action":"rename"}]`); err != nil {
		t.Fatalf("Set upload.dangerous_extension_rules: %v", err)
	}

	body := []byte("same-body")
	legacy := newDangerousUploadTestFile("legacy-unsafe", "u1", body)
	if err := svc.fileRepo.Create(context.Background(), legacy); err != nil {
		t.Fatalf("seed legacy file: %v", err)
	}

	result, err := svc.Upload(context.Background(), UploadParams{
		UserID:   "u1",
		Filename: "setup.exe",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if result.File.ID == legacy.ID {
		t.Fatal("renamed upload should not reuse unsafe historical record")
	}
	if result.File.OriginalName != "setup.exe.blocked" {
		t.Fatalf("unexpected original name: %q", result.File.OriginalName)
	}
}

func TestFileService_UploadRenamedDangerousExtensionReusesSafeDuplicate(t *testing.T) {
	svc, cleanup, settingRepo := newUploadPathTestService(t)
	defer cleanup()

	if err := settingRepo.Set(context.Background(), UploadDangerousExtensionRulesSettingKey, `[{"ext":".exe","action":"rename"}]`); err != nil {
		t.Fatalf("Set upload.dangerous_extension_rules: %v", err)
	}

	body := []byte("same-body-safe")
	first, err := svc.Upload(context.Background(), UploadParams{
		UserID:   "u1",
		Filename: "setup.exe",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("first Upload: %v", err)
	}

	second, err := svc.Upload(context.Background(), UploadParams{
		UserID:   "u1",
		Filename: "another.exe",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("second Upload: %v", err)
	}

	if second.File.ID != first.File.ID {
		t.Fatalf("expected safe duplicate reuse, got first=%q second=%q", first.File.ID, second.File.ID)
	}
}

func newDangerousUploadTestFile(id, userID string, body []byte) *model.File {
	hash := md5.Sum(body)
	hashMD5 := hex.EncodeToString(hash[:])
	return &model.File{
		ID:              id,
		UserID:          userID,
		StorageConfigID: "local-1",
		OriginalName:    "legacy.exe",
		StorageKey:      "legacy/legacy.exe",
		HashMD5:         hashMD5,
		SizeBytes:       int64(len(body)),
		MimeType:        "application/x-msdownload",
		FileType:        model.FileTypeFile,
		URL:             "/f/" + hashMD5[:8],
		CreatedAt:       time.Now().Add(-time.Hour),
	}
}
