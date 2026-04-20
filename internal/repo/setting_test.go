package repo

import (
	"context"
	"errors"
	"testing"

	"github.com/amigoer/kite/internal/model"
	"gorm.io/gorm"
)

func TestSettingRepo_GetMissing(t *testing.T) {
	r := NewSettingRepo(newTestDB(t))
	_, err := r.Get(context.Background(), "nonexistent-key")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Logf("error is: %v (not gorm.ErrRecordNotFound, which is OK if wrapped)", err)
	}
}

func TestSettingRepo_SetAndGet(t *testing.T) {
	r := NewSettingRepo(newTestDB(t))
	ctx := context.Background()

	if err := r.Set(ctx, "site_name", "Kite"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := r.Get(ctx, "site_name")
	if err != nil || val != "Kite" {
		t.Fatalf("Get after Set: err=%v val=%q", err, val)
	}

	// Upsert: overwrite existing value.
	if err := r.Set(ctx, "site_name", "Kite v2"); err != nil {
		t.Fatalf("Set upsert: %v", err)
	}
	val, _ = r.Get(ctx, "site_name")
	if val != "Kite v2" {
		t.Fatalf("Upsert not applied: %q", val)
	}
}

func TestSettingRepo_GetAll(t *testing.T) {
	r := NewSettingRepo(newTestDB(t))
	ctx := context.Background()

	r.Set(ctx, "key1", "val1")
	r.Set(ctx, "key2", "val2")

	all, err := r.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if all["key1"] != "val1" || all["key2"] != "val2" {
		t.Fatalf("GetAll contents wrong: %v", all)
	}
}

func TestSettingRepo_SetBatch(t *testing.T) {
	r := NewSettingRepo(newTestDB(t))
	ctx := context.Background()

	batch := map[string]string{
		"upload_max_size": "104857600",
		"allow_register":  "true",
		"default_quota":   "10737418240",
	}
	if err := r.SetBatch(ctx, batch); err != nil {
		t.Fatalf("SetBatch: %v", err)
	}

	for k, want := range batch {
		got, err := r.Get(ctx, k)
		if err != nil || got != want {
			t.Fatalf("key %q: err=%v got=%q want=%q", k, err, got, want)
		}
	}
}

func TestSettingRepo_Delete(t *testing.T) {
	r := NewSettingRepo(newTestDB(t))
	ctx := context.Background()

	r.Set(ctx, "temp_key", "temp_val")

	if err := r.Delete(ctx, "temp_key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := r.Get(ctx, "temp_key")
	if err == nil {
		t.Fatal("key should be deleted")
	}
}

func TestSetting_TableName(t *testing.T) {
	if (model.Setting{}).TableName() != "settings" {
		t.Fatal("unexpected table name")
	}
}
