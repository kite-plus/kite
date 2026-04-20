package repo

import (
	"context"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/model"
)

func TestFileAccessLogRepo_Create(t *testing.T) {
	r := NewFileAccessLogRepo(newTestDB(t))
	ctx := context.Background()

	log := &model.FileAccessLog{
		ID:          "fal1",
		FileID:      "f1",
		UserID:      "u1",
		BytesServed: 1024,
		AccessedAt:  time.Now(),
	}
	if err := r.Create(ctx, log); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestFileAccessLogRepo_GetDailyAccessStats_UserScoped(t *testing.T) {
	r := NewFileAccessLogRepo(newTestDB(t))
	ctx := context.Background()

	start := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)

	// In-window log for u1.
	r.Create(ctx, &model.FileAccessLog{
		ID: "fal2", FileID: "f1", UserID: "u1",
		BytesServed: 512, AccessedAt: start.Add(time.Hour),
	})
	// In-window log for u2 — should not appear in u1's stats.
	r.Create(ctx, &model.FileAccessLog{
		ID: "fal3", FileID: "f2", UserID: "u2",
		BytesServed: 999, AccessedAt: start.Add(2 * time.Hour),
	})
	// Out-of-window log for u1.
	r.Create(ctx, &model.FileAccessLog{
		ID: "fal4", FileID: "f1", UserID: "u1",
		BytesServed: 256, AccessedAt: start.Add(-time.Hour),
	})

	rows, err := r.GetDailyAccessStats(ctx, "u1", start, end)
	if err != nil {
		t.Fatalf("GetDailyAccessStats: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 day row, got %d", len(rows))
	}
	if rows[0].AccessCount != 1 || rows[0].BytesServed != 512 {
		t.Fatalf("unexpected stats: %+v", rows[0])
	}
}

func TestFileAccessLogRepo_GetDailyAccessStats_AdminView(t *testing.T) {
	r := NewFileAccessLogRepo(newTestDB(t))
	ctx := context.Background()

	start := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)

	r.Create(ctx, &model.FileAccessLog{
		ID: "adm1", FileID: "f1", UserID: "u1",
		BytesServed: 100, AccessedAt: start.Add(time.Hour),
	})
	r.Create(ctx, &model.FileAccessLog{
		ID: "adm2", FileID: "f2", UserID: "u2",
		BytesServed: 200, AccessedAt: start.Add(2 * time.Hour),
	})

	rows, err := r.GetDailyAccessStats(ctx, "", start, end) // empty userID = admin
	if err != nil {
		t.Fatalf("GetDailyAccessStats admin: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 day row, got %d", len(rows))
	}
	if rows[0].AccessCount != 2 || rows[0].BytesServed != 300 {
		t.Fatalf("unexpected admin stats: %+v", rows[0])
	}
}

func TestFileAccessLogRepo_GetHourlyAccessHeatmapStats(t *testing.T) {
	r := NewFileAccessLogRepo(newTestDB(t))
	ctx := context.Background()

	start := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)

	r.Create(ctx, &model.FileAccessLog{
		ID: "hh1", FileID: "f1", UserID: "u1",
		BytesServed: 1, AccessedAt: time.Date(2025, 5, 2, 9, 0, 0, 0, time.UTC),
	})

	rows, err := r.GetHourlyAccessHeatmapStats(ctx, "u1", start, end)
	if err != nil {
		t.Fatalf("GetHourlyAccessHeatmapStats: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(rows))
	}
	if rows[0].Hour != 9 {
		t.Fatalf("expected hour 9, got %d", rows[0].Hour)
	}
	if rows[0].Count != 1 {
		t.Fatalf("expected count 1, got %d", rows[0].Count)
	}
}
