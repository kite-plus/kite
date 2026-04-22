package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
)

func TestRespondUploadError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		err        error
		isGuest    bool
		wantStatus int
		wantCode   int
		wantMsg    string
	}{
		{
			name:       "文件大小超限",
			err:        fmt.Errorf("wrapped: %w", service.ErrFileTooLarge),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   41300,
			wantMsg:    uploadFileTooLargeMessage,
		},
		{
			name:       "文件类型不允许",
			err:        fmt.Errorf("wrapped: %w", service.ErrFileTypeDenied),
			wantStatus: http.StatusUnsupportedMediaType,
			wantCode:   41500,
			wantMsg:    uploadFileTypeDeniedMessage,
		},
		{
			name:       "存储空间不足",
			err:        fmt.Errorf("wrapped: %w", service.ErrStorageFull),
			wantStatus: http.StatusInsufficientStorage,
			wantCode:   50700,
			wantMsg:    uploadStorageFullMessage,
		},
		{
			name:       "登录上传默认错误",
			err:        errors.New("put file failed"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   50000,
			wantMsg:    uploadSaveFailedMessage,
		},
		{
			name:       "游客上传默认错误",
			err:        errors.New("put file failed"),
			isGuest:    true,
			wantStatus: http.StatusInternalServerError,
			wantCode:   50000,
			wantMsg:    guestUploadFailedMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)

			respondUploadError(c, tt.err, tt.isGuest)

			if rec.Code != tt.wantStatus {
				t.Fatalf("unexpected status: got=%d want=%d", rec.Code, tt.wantStatus)
			}

			var payload Response
			if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Code != tt.wantCode {
				t.Fatalf("unexpected code: got=%d want=%d", payload.Code, tt.wantCode)
			}
			if payload.Message != tt.wantMsg {
				t.Fatalf("unexpected message: got=%q want=%q", payload.Message, tt.wantMsg)
			}
		})
	}
}

func TestFileHandler_UploadRequiresFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Handlers call fileSvc.MaxUploadBodySize before FormFile to wrap the
	// body in MaxBytesReader. A bare &FileHandler{} with a nil service would
	// panic on that method call, so hand over a zero-value FileService —
	// nil settingRepo falls back to cfg.MaxFileSize, and the empty request
	// body in this test cannot exceed any cap.
	svc := &service.FileService{}
	fh := &FileHandler{fileSvc: svc}

	tests := []struct {
		name string
		path string
		h    gin.HandlerFunc
	}{
		{
			name: "登录上传缺少文件",
			path: "/upload",
			h:    fh.Upload,
		},
		{
			name: "游客上传缺少文件",
			path: "/guest/upload",
			h:    fh.GuestUpload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST(tt.path, tt.h)

			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("unexpected status: got=%d want=%d", rec.Code, http.StatusBadRequest)
			}

			var payload Response
			if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload.Message != uploadMissingFileMessage {
				t.Fatalf("unexpected message: got=%q want=%q", payload.Message, uploadMissingFileMessage)
			}
		})
	}
}
