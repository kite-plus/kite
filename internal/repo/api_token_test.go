package repo

import (
	"context"
	"testing"
	"time"

	"github.com/amigoer/kite/internal/model"
)

func TestAPITokenRepo_CreateAndGetByHash(t *testing.T) {
	r := NewAPITokenRepo(newTestDB(t))
	ctx := context.Background()

	tok := &model.APIToken{ID: "t1", UserID: "u1", Name: "PicGo", TokenHash: "hash001"}
	if err := r.Create(ctx, tok); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.GetByTokenHash(ctx, "hash001")
	if err != nil || got.ID != "t1" {
		t.Fatalf("GetByTokenHash: err=%v got=%v", err, got)
	}
}

func TestAPITokenRepo_GetByTokenHash_NotFound(t *testing.T) {
	r := NewAPITokenRepo(newTestDB(t))
	_, err := r.GetByTokenHash(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestAPITokenRepo_ListByUser(t *testing.T) {
	r := NewAPITokenRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.APIToken{ID: "t2", UserID: "u1", Name: "A", TokenHash: "h2"})
	r.Create(ctx, &model.APIToken{ID: "t3", UserID: "u1", Name: "B", TokenHash: "h3"})
	r.Create(ctx, &model.APIToken{ID: "t4", UserID: "u2", Name: "C", TokenHash: "h4"})

	list, err := r.ListByUser(ctx, "u1")
	if err != nil || len(list) != 2 {
		t.Fatalf("ListByUser: err=%v len=%d", err, len(list))
	}
}

func TestAPITokenRepo_Delete(t *testing.T) {
	r := NewAPITokenRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.APIToken{ID: "t5", UserID: "u1", Name: "Del", TokenHash: "h5"})

	if err := r.Delete(ctx, "t5", "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := r.GetByTokenHash(ctx, "h5")
	if err == nil {
		t.Fatal("token should be deleted")
	}
}

func TestAPITokenRepo_Delete_WrongUser(t *testing.T) {
	r := NewAPITokenRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.APIToken{ID: "t6", UserID: "u1", Name: "X", TokenHash: "h6"})

	// Deleting another user's token should fail silently (ErrRecordNotFound equivalent).
	err := r.Delete(ctx, "t6", "other-user")
	if err == nil {
		t.Fatal("expected error when deleting another user's token")
	}
}

func TestAPITokenRepo_UpdateLastUsed(t *testing.T) {
	r := NewAPITokenRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.APIToken{ID: "t7", UserID: "u1", Name: "Y", TokenHash: "h7"})

	if err := r.UpdateLastUsed(ctx, "t7"); err != nil {
		t.Fatalf("UpdateLastUsed: %v", err)
	}

	got, _ := r.GetByTokenHash(ctx, "h7")
	if got.LastUsed == nil {
		t.Fatal("LastUsed should be set after UpdateLastUsed")
	}
	if got.LastUsed.IsZero() {
		t.Fatal("LastUsed should not be zero time")
	}
}

func TestAPIToken_IsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	expired := &model.APIToken{ExpiresAt: &past}
	if !expired.IsExpired() {
		t.Fatal("past expiry should be expired")
	}

	valid := &model.APIToken{ExpiresAt: &future}
	if valid.IsExpired() {
		t.Fatal("future expiry should not be expired")
	}

	noExpiry := &model.APIToken{}
	if noExpiry.IsExpired() {
		t.Fatal("nil ExpiresAt should never be expired")
	}
}
