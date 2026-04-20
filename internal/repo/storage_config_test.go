package repo

import (
	"context"
	"testing"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

func makeStorageConfig(id, name string, priority int, isDefault, isActive bool) *model.StorageConfig {
	return &model.StorageConfig{
		ID:        id,
		Name:      name,
		Driver:    "local",
		Config:    `{"base_path":"/tmp","base_url":"http://localhost"}`,
		Priority:  priority,
		IsDefault: isDefault,
		IsActive:  isActive,
	}
}

// createStorageConfig inserts the config and then explicitly sets boolean fields that are false,
// working around GORM's behavior of using DB defaults for zero-value booleans.
func createStorageConfig(t *testing.T, db *gorm.DB, r *StorageConfigRepo, cfg *model.StorageConfig) {
	t.Helper()
	// Create with IsActive and IsDefault forced to true first (so GORM doesn't drop them),
	// then use raw updates to set the intended values.
	origActive := cfg.IsActive
	origDefault := cfg.IsDefault
	cfg.IsActive = true
	cfg.IsDefault = true
	if err := r.Create(context.Background(), cfg); err != nil {
		t.Fatalf("createStorageConfig: %v", err)
	}
	// Now correct the booleans if they differ from the defaults.
	if !origActive {
		db.Model(&model.StorageConfig{}).Where("id = ?", cfg.ID).Update("is_active", false)
	}
	if !origDefault {
		db.Model(&model.StorageConfig{}).Where("id = ?", cfg.ID).Update("is_default", false)
	}
	cfg.IsActive = origActive
	cfg.IsDefault = origDefault
}

func TestStorageConfigRepo_CreateAndGetByID(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	cfg := makeStorageConfig("sc1", "Local", 100, false, true)
	createStorageConfig(t, db, r, cfg)

	got, err := r.GetByID(ctx, "sc1")
	if err != nil || got.Name != "Local" {
		t.Fatalf("GetByID: err=%v got=%v", err, got)
	}
}

func TestStorageConfigRepo_GetByID_NotFound(t *testing.T) {
	r := NewStorageConfigRepo(newTestDB(t))
	_, err := r.GetByID(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestStorageConfigRepo_GetDefault(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("sc2", "Primary", 100, true, true))
	createStorageConfig(t, db, r, makeStorageConfig("sc3", "Secondary", 200, false, true))

	got, err := r.GetDefault(ctx)
	if err != nil || got.ID != "sc2" {
		t.Fatalf("GetDefault: err=%v got=%v", err, got)
	}
}

func TestStorageConfigRepo_GetDefault_InactiveIgnored(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	// is_default=true but is_active=false — should not be returned.
	createStorageConfig(t, db, r, makeStorageConfig("sc4", "Inactive Default", 100, true, false))

	_, err := r.GetDefault(ctx)
	if err == nil {
		t.Fatal("inactive default should not be returned")
	}
}

func TestStorageConfigRepo_SetDefault(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("sc5", "A", 100, true, true))
	createStorageConfig(t, db, r, makeStorageConfig("sc6", "B", 200, false, true))

	if err := r.SetDefault(ctx, "sc6"); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}

	newDefault, _ := r.GetDefault(ctx)
	if newDefault.ID != "sc6" {
		t.Fatalf("new default should be sc6, got %s", newDefault.ID)
	}

	// Old default should be cleared.
	old, _ := r.GetByID(ctx, "sc5")
	if old.IsDefault {
		t.Fatal("old default should be cleared")
	}
}

func TestStorageConfigRepo_List_Order(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("lo1", "C", 300, false, true))
	createStorageConfig(t, db, r, makeStorageConfig("lo2", "A", 100, false, true))
	createStorageConfig(t, db, r, makeStorageConfig("lo3", "B", 200, false, true))

	list, err := r.List(ctx)
	if err != nil || len(list) != 3 {
		t.Fatalf("List: err=%v len=%d", err, len(list))
	}
	if list[0].ID != "lo2" || list[1].ID != "lo3" || list[2].ID != "lo1" {
		t.Fatalf("List order wrong: %v %v %v", list[0].ID, list[1].ID, list[2].ID)
	}
}

func TestStorageConfigRepo_ListActive(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("la1", "Active", 100, false, true))
	createStorageConfig(t, db, r, makeStorageConfig("la2", "Inactive", 200, false, false))

	list, err := r.ListActive(ctx)
	if err != nil || len(list) != 1 || list[0].ID != "la1" {
		t.Fatalf("ListActive: err=%v list=%v", err, list)
	}
}

func TestStorageConfigRepo_Reorder(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("ro1", "X", 100, false, true))
	createStorageConfig(t, db, r, makeStorageConfig("ro2", "Y", 200, false, true))
	createStorageConfig(t, db, r, makeStorageConfig("ro3", "Z", 300, false, true))

	// Reverse the order.
	if err := r.Reorder(ctx, []string{"ro3", "ro2", "ro1"}); err != nil {
		t.Fatalf("Reorder: %v", err)
	}

	list, _ := r.List(ctx)
	if list[0].ID != "ro3" || list[1].ID != "ro2" || list[2].ID != "ro1" {
		t.Fatalf("Reorder result wrong: %v %v %v", list[0].ID, list[1].ID, list[2].ID)
	}
	if list[0].Priority != 100 || list[1].Priority != 200 || list[2].Priority != 300 {
		t.Fatalf("Priorities wrong: %d %d %d", list[0].Priority, list[1].Priority, list[2].Priority)
	}
}

func TestStorageConfigRepo_BuildRawConfigs(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("br1", "Bucket1", 100, true, true))
	createStorageConfig(t, db, r, makeStorageConfig("br2", "Bucket2", 200, false, false))

	raws, err := r.BuildRawConfigs(ctx)
	if err != nil || len(raws) != 2 {
		t.Fatalf("BuildRawConfigs: err=%v len=%d", err, len(raws))
	}

	// Verify first entry maps correctly.
	if raws[0].ID != "br1" || raws[0].Name != "Bucket1" || !raws[0].IsDefault || !raws[0].IsActive {
		t.Fatalf("BuildRawConfigs entry[0]: %+v", raws[0])
	}
	if raws[1].IsActive {
		t.Fatal("BuildRawConfigs entry[1] should have IsActive=false")
	}
}

func TestStorageConfigRepo_Delete(t *testing.T) {
	db := newTestDB(t)
	r := NewStorageConfigRepo(db)
	ctx := context.Background()

	createStorageConfig(t, db, r, makeStorageConfig("del1", "Delete Me", 100, false, true))

	if err := r.Delete(ctx, "del1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := r.GetByID(ctx, "del1")
	if err == nil {
		t.Fatal("config should be deleted")
	}
}
