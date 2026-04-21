package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestFileService_UploadUsesRuntimeMaxFileSize(t *testing.T) {
	svc, cleanup, settingRepo := newUploadPathTestService(t)
	defer cleanup()

	if err := settingRepo.Set(context.Background(), UploadMaxFileSizeMBSettingKey, "1"); err != nil {
		t.Fatalf("Set upload.max_file_size_mb: %v", err)
	}

	body := bytes.Repeat([]byte("a"), 1024*1024+1)
	_, err := svc.Upload(context.Background(), UploadParams{
		UserID:   "u1",
		Filename: "too-large.txt",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}
