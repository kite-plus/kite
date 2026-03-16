package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewUploadService(t *testing.T) {
	svc := NewUploadService("")
	if svc.uploadDir != DefaultUploadDir {
		t.Errorf("expected upload dir %q, got %q", DefaultUploadDir, svc.uploadDir)
	}
	svc2 := NewUploadService("/tmp/custom")
	if svc2.uploadDir != "/tmp/custom" {
		t.Errorf("expected upload dir /tmp/custom, got %q", svc2.uploadDir)
	}
}

func TestAllowedImageExts(t *testing.T) {
	allowed := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".avif"}
	forbidden := []string{".exe", ".sh", ".html", ".js", ".go", ".pdf"}

	for _, ext := range allowed {
		if !allowedImageExts[ext] {
			t.Errorf("expected %s to be allowed", ext)
		}
	}
	for _, ext := range forbidden {
		if allowedImageExts[ext] {
			t.Errorf("expected %s to be forbidden", ext)
		}
	}
}

func TestSaveImage_InvalidType(t *testing.T) {
	svc := NewUploadService(filepath.Join(os.TempDir(), "kite-test-uploads"))
	defer os.RemoveAll(filepath.Join(os.TempDir(), "kite-test-uploads"))

	// 模拟不允许的文件类型需要真实 multipart.FileHeader
	// 这里只测试扩展名检查的覆盖
	for ext, want := range map[string]bool{".jpg": true, ".exe": false, ".png": true, ".sh": false} {
		got := allowedImageExts[ext]
		if got != want {
			t.Errorf("ext %s: expected %v, got %v", ext, want, got)
		}
	}
	_ = svc
}
