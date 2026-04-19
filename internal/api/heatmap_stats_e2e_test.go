package api

import (
	"encoding/json"
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

func TestHeatmapStatsEndpoints_E2E(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.File{}, &model.FileAccessLog{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}

	fileRepo := repo.NewFileRepo(db)
	accessRepo := repo.NewFileAccessLogRepo(db)
	h := NewUserHandler(repo.NewUserRepo(db), fileRepo, accessRepo, nil)

	// Monday 10:00 UTC
	ts := time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC)

	mustCreateFile(t, db, &model.File{
		ID:              "f-u1",
		UserID:          "u1",
		StorageConfigID: "s1",
		OriginalName:    "u1.jpg",
		StorageKey:      "2026/04/u1.jpg",
		HashMD5:         "11111111aaaaaaaa11111111aaaaaaaa",
		SizeBytes:       123,
		MimeType:        "image/jpeg",
		FileType:        model.FileTypeImage,
		URL:             "http://localhost/i/11111111",
		CreatedAt:       ts,
	})
	mustCreateFile(t, db, &model.File{
		ID:              "f-u2",
		UserID:          "u2",
		StorageConfigID: "s1",
		OriginalName:    "u2.jpg",
		StorageKey:      "2026/04/u2.jpg",
		HashMD5:         "22222222aaaaaaaa22222222aaaaaaaa",
		SizeBytes:       456,
		MimeType:        "image/jpeg",
		FileType:        model.FileTypeImage,
		URL:             "http://localhost/i/22222222",
		CreatedAt:       ts,
	})

	mustCreateAccess(t, db, &model.FileAccessLog{ID: "a1", FileID: "f-u1", UserID: "u1", BytesServed: 100, AccessedAt: ts})
	mustCreateAccess(t, db, &model.FileAccessLog{ID: "a2", FileID: "f-u1", UserID: "u1", BytesServed: 100, AccessedAt: ts})

	r := gin.New()
	r.GET("/api/v1/stats/heatmap", func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, "u1")
		h.HeatmapStats(c)
	})
	r.GET("/api/v1/admin/stats/heatmap", h.AdminHeatmapStats)

	tsrv := httptest.NewServer(r)
	defer tsrv.Close()

	userResp := getHeatmapResponse(t, tsrv.URL+"/api/v1/stats/heatmap?weeks=12")
	adminResp := getHeatmapResponse(t, tsrv.URL+"/api/v1/admin/stats/heatmap?weeks=12")

	if userResp.Code != 0 {
		t.Fatalf("user heatmap code=%d", userResp.Code)
	}
	if adminResp.Code != 0 {
		t.Fatalf("admin heatmap code=%d", adminResp.Code)
	}

	if len(userResp.Data.Grid) != 7 || len(userResp.Data.Grid[0]) != 24 {
		t.Fatalf("invalid user heatmap grid shape: %dx%d", len(userResp.Data.Grid), len(userResp.Data.Grid[0]))
	}

	// Monday maps to row 0, hour 10 maps to col 10.
	if got := userResp.Data.Grid[0][10]; got != 3 {
		t.Fatalf("user heatmap monday-10 expected 3, got %d", got)
	}
	if got := adminResp.Data.Grid[0][10]; got != 4 {
		t.Fatalf("admin heatmap monday-10 expected 4, got %d", got)
	}
}

type heatmapResponse struct {
	Code int `json:"code"`
	Data struct {
		Weeks int       `json:"weeks"`
		Grid  [][]int64 `json:"grid"`
	} `json:"data"`
}

func getHeatmapResponse(t *testing.T, url string) heatmapResponse {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get heatmap failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heatmap status=%d", resp.StatusCode)
	}

	var payload heatmapResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode heatmap response failed: %v", err)
	}
	return payload
}

func mustCreateFile(t *testing.T, db *gorm.DB, f *model.File) {
	t.Helper()
	if err := db.Create(f).Error; err != nil {
		t.Fatalf("create file failed: %v", err)
	}
}

func mustCreateAccess(t *testing.T, db *gorm.DB, a *model.FileAccessLog) {
	t.Helper()
	if err := db.Create(a).Error; err != nil {
		t.Fatalf("create access failed: %v", err)
	}
}
