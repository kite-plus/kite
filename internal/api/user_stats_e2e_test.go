package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/api/middleware"
	"github.com/amigoer/kite/internal/model"
	"github.com/amigoer/kite/internal/repo"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUserStats_ReturnsPerTypeSizeMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:user-stats-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.File{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	seed := []model.File{
		{
			ID:              "img-1",
			UserID:          "u1",
			StorageConfigID: "s1",
			OriginalName:    "a.jpg",
			StorageKey:      "a.jpg",
			HashMD5:         "11111111aaaaaaaa11111111aaaaaaaa",
			SizeBytes:       100,
			MimeType:        "image/jpeg",
			FileType:        model.FileTypeImage,
			URL:             "/i/11111111",
			CreatedAt:       time.Now(),
		},
		{
			ID:              "vid-1",
			UserID:          "u1",
			StorageConfigID: "s1",
			OriginalName:    "b.mp4",
			StorageKey:      "b.mp4",
			HashMD5:         "22222222aaaaaaaa22222222aaaaaaaa",
			SizeBytes:       200,
			MimeType:        "video/mp4",
			FileType:        model.FileTypeVideo,
			URL:             "/v/22222222",
			CreatedAt:       time.Now(),
		},
		{
			ID:              "aud-1",
			UserID:          "u1",
			StorageConfigID: "s1",
			OriginalName:    "c.mp3",
			StorageKey:      "c.mp3",
			HashMD5:         "33333333aaaaaaaa33333333aaaaaaaa",
			SizeBytes:       300,
			MimeType:        "audio/mpeg",
			FileType:        model.FileTypeAudio,
			URL:             "/a/33333333",
			CreatedAt:       time.Now(),
		},
		{
			ID:              "file-1",
			UserID:          "u1",
			StorageConfigID: "s1",
			OriginalName:    "d.zip",
			StorageKey:      "d.zip",
			HashMD5:         "44444444aaaaaaaa44444444aaaaaaaa",
			SizeBytes:       400,
			MimeType:        "application/zip",
			FileType:        model.FileTypeFile,
			URL:             "/f/44444444",
			CreatedAt:       time.Now(),
		},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed file failed: %v", err)
		}
	}

	h := NewUserHandler(repo.NewUserRepo(db), repo.NewFileRepo(db), repo.NewFileAccessLogRepo(db), nil)

	r := gin.New()
	r.GET("/api/v1/stats", func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "u1")
		h.Stats(c)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/stats")
	if err != nil {
		t.Fatalf("request stats failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status=%d", resp.StatusCode)
	}

	var payload struct {
		Code int `json:"code"`
		Data struct {
			ImageSize int64 `json:"images_size"`
			VideoSize int64 `json:"videos_size"`
			AudioSize int64 `json:"audios_size"`
			OtherSize int64 `json:"others_size"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode stats response failed: %v", err)
	}

	if payload.Code != 0 {
		t.Fatalf("unexpected code: %d", payload.Code)
	}
	if payload.Data.ImageSize != 100 || payload.Data.VideoSize != 200 || payload.Data.AudioSize != 300 || payload.Data.OtherSize != 400 {
		t.Fatalf("unexpected per-type sizes: %+v", payload.Data)
	}
}
