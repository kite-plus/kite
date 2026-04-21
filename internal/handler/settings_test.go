package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/amigoer/kite/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var settingsHandlerTestCounter int64

func TestSettingsHandler_GetIncludesDefaultUploadPathPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSettingsHandlerTestDB(t)
	h := NewSettingsHandler(
		repo.NewSettingRepo(db),
		service.DefaultSettings("Kite", "http://localhost:8080", true, "{year}/{month}/{md5_8}/{uuid}.{ext}", 100*1024*1024),
	)

	r := gin.New()
	r.GET("/settings", h.Get)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /settings status=%d", rec.Code)
	}

	var payload struct {
		Code int               `json:"code"`
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := payload.Data[service.UploadPathPatternSettingKey]; got != "{year}/{month}/{md5_8}/{uuid}.{ext}" {
		t.Fatalf("unexpected default upload path pattern: %q", got)
	}
	if got := payload.Data[service.UploadMaxFileSizeMBSettingKey]; got != "100" {
		t.Fatalf("unexpected default upload max size: %q", got)
	}
	if got := payload.Data[service.SiteTitleSettingKey]; got != "Kite - 自部署媒体托管系统" {
		t.Fatalf("unexpected default site title: %q", got)
	}
	if got := payload.Data[service.SiteFaviconURLSettingKey]; got != "/favicon.svg" {
		t.Fatalf("unexpected default favicon url: %q", got)
	}
}

func TestSettingsHandler_UpdateAcceptsValidUploadMaxFileSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSettingsHandlerTestDB(t)
	settingRepo := repo.NewSettingRepo(db)
	h := NewSettingsHandler(settingRepo, nil)

	r := gin.New()
	r.PUT("/settings", h.Update)

	body := map[string]any{
		"settings": map[string]string{
			service.UploadMaxFileSizeMBSettingKey: " 256 ",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /settings status=%d body=%s", rec.Code, rec.Body.String())
	}

	saved, err := settingRepo.Get(req.Context(), service.UploadMaxFileSizeMBSettingKey)
	if err != nil {
		t.Fatalf("Get upload.max_file_size_mb: %v", err)
	}
	if saved != "256" {
		t.Fatalf("unexpected normalized upload max size: %q", saved)
	}
}

func TestSettingsHandler_UpdateRejectsInvalidUploadMaxFileSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSettingsHandlerTestDB(t)
	h := NewSettingsHandler(repo.NewSettingRepo(db), nil)

	r := gin.New()
	r.PUT("/settings", h.Update)

	body := map[string]any{
		"settings": map[string]string{
			service.UploadMaxFileSizeMBSettingKey: "0",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid upload.max_file_size_mb, got %d", rec.Code)
	}
}

func TestSettingsHandler_UpdateAcceptsValidUploadPathPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSettingsHandlerTestDB(t)
	settingRepo := repo.NewSettingRepo(db)
	h := NewSettingsHandler(settingRepo, nil)

	r := gin.New()
	r.PUT("/settings", h.Update)

	body := map[string]any{
		"settings": map[string]string{
			service.UploadPathPatternSettingKey: "{user_id}/{uuid}.{ext}/",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /settings status=%d body=%s", rec.Code, rec.Body.String())
	}

	saved, err := settingRepo.Get(req.Context(), service.UploadPathPatternSettingKey)
	if err != nil {
		t.Fatalf("Get upload.path_pattern: %v", err)
	}
	if saved != "{user_id}/{uuid}.{ext}" {
		t.Fatalf("unexpected normalized pattern: %q", saved)
	}
}

func TestSettingsHandler_UpdateRejectsInvalidUploadPathPattern(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSettingsHandlerTestDB(t)
	h := NewSettingsHandler(repo.NewSettingRepo(db), nil)

	r := gin.New()
	r.PUT("/settings", h.Update)

	body := map[string]any{
		"settings": map[string]string{
			service.UploadPathPatternSettingKey: "/{year}/{uuid}.{ext}",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pattern, got %d", rec.Code)
	}

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message == "" {
		t.Fatal("expected error message for invalid upload.path_pattern")
	}
}

func TestSettingsHandler_UpdateRejectsEmptySiteTitle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newSettingsHandlerTestDB(t)
	h := NewSettingsHandler(repo.NewSettingRepo(db), nil)

	r := gin.New()
	r.PUT("/settings", h.Update)

	body := map[string]any{
		"settings": map[string]string{
			service.SiteTitleSettingKey: "   ",
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty site_title, got %d", rec.Code)
	}
}

func newSettingsHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	id := atomic.AddInt64(&settingsHandlerTestCounter, 1)
	dsn := fmt.Sprintf("file:settings-handler-test-%d?mode=memory&cache=shared", id)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatalf("migrate settings: %v", err)
	}
	return db
}
