package repo

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/amigoer/kite/internal/model"
)

func TestUserRepo_CreateAndGetByID(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "u1", Username: "alice", Email: "alice@example.com", PasswordHash: "hash", Role: "user"}
	if err := r.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Username != "alice" || got.Email != "alice@example.com" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	_, err := r.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestUserRepo_GetByUsername(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "u2", Username: "bob", Email: "bob@example.com", PasswordHash: "h", Role: "user"}
	r.Create(ctx, u)

	got, err := r.GetByUsername(ctx, "bob")
	if err != nil || got.ID != "u2" {
		t.Fatalf("GetByUsername: err=%v got=%v", err, got)
	}

	_, err = r.GetByUsername(ctx, "nobody")
	if err == nil {
		t.Fatal("expected error for missing username")
	}
}

func TestUserRepo_GetByEmail(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "u3", Username: "carol", Email: "carol@example.com", PasswordHash: "h", Role: "user"}
	r.Create(ctx, u)

	got, err := r.GetByEmail(ctx, "carol@example.com")
	if err != nil || got.ID != "u3" {
		t.Fatalf("GetByEmail: %v", err)
	}

	_, err = r.GetByEmail(ctx, "ghost@example.com")
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestUserRepo_Update(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	nick := "Alice"
	u := &model.User{ID: "u4", Username: "alice2", Email: "alice2@example.com", PasswordHash: "h", Role: "user"}
	r.Create(ctx, u)

	u.Nickname = &nick
	if err := r.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := r.GetByID(ctx, "u4")
	if got.Nickname == nil || *got.Nickname != "Alice" {
		t.Fatalf("nickname not persisted: %v", got.Nickname)
	}
}

func TestUserRepo_UpdateStorageUsed(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "u5", Username: "dave", Email: "dave@example.com", PasswordHash: "h", Role: "user", StorageUsed: 0}
	r.Create(ctx, u)

	if err := r.UpdateStorageUsed(ctx, "u5", 1024); err != nil {
		t.Fatalf("UpdateStorageUsed positive: %v", err)
	}
	got, _ := r.GetByID(ctx, "u5")
	if got.StorageUsed != 1024 {
		t.Fatalf("expected 1024, got %d", got.StorageUsed)
	}

	if err := r.UpdateStorageUsed(ctx, "u5", -512); err != nil {
		t.Fatalf("UpdateStorageUsed negative: %v", err)
	}
	got, _ = r.GetByID(ctx, "u5")
	if got.StorageUsed != 512 {
		t.Fatalf("expected 512, got %d", got.StorageUsed)
	}
}

func TestUserRepo_Delete(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "u6", Username: "eve", Email: "eve@example.com", PasswordHash: "h", Role: "user", IsActive: true}
	r.Create(ctx, u)

	if err := r.Delete(ctx, "u6"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Row still exists in DB but is_active should be false.
	var found model.User
	if err := r.db.Where("id = ?", "u6").First(&found).Error; err != nil {
		t.Fatalf("user row should still exist: %v", err)
	}
	if found.IsActive {
		t.Fatal("expected is_active=false after delete")
	}
}

func TestUserRepo_List_Pagination(t *testing.T) {
	db := newTestDB(t)
	r := NewUserRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		u := &model.User{
			ID:           fmt.Sprintf("ul%d", i),
			Username:     fmt.Sprintf("user%d", i),
			Email:        fmt.Sprintf("user%d@example.com", i),
			PasswordHash: "h",
			Role:         "user",
		}
		r.Create(ctx, u)
	}

	page1, total, err := r.List(ctx, 1, 3)
	if err != nil || total != 5 || len(page1) != 3 {
		t.Fatalf("List page1: err=%v total=%d len=%d", err, total, len(page1))
	}

	page2, _, err := r.List(ctx, 2, 3)
	if err != nil || len(page2) != 2 {
		t.Fatalf("List page2: err=%v len=%d", err, len(page2))
	}
}

func TestUserRepo_ExistsByUsernameOrEmail(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "ue1", Username: "frank", Email: "frank@example.com", PasswordHash: "h", Role: "user"}
	r.Create(ctx, u)

	exists, err := r.ExistsByUsernameOrEmail(ctx, "frank", "nobody@example.com")
	if err != nil || !exists {
		t.Fatalf("expected exists=true on username match: err=%v exists=%v", err, exists)
	}

	exists, _ = r.ExistsByUsernameOrEmail(ctx, "nobody", "frank@example.com")
	if !exists {
		t.Fatal("expected exists=true on email match")
	}

	exists, _ = r.ExistsByUsernameOrEmail(ctx, "nobody", "nobody@example.com")
	if exists {
		t.Fatal("expected exists=false for no match")
	}
}

func TestUserRepo_ExistsByUsernameOrEmailExcept(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "ue2", Username: "grace", Email: "grace@example.com", PasswordHash: "h", Role: "user"}
	r.Create(ctx, u)

	// Editing own profile — should not conflict with itself.
	exists, err := r.ExistsByUsernameOrEmailExcept(ctx, "grace", "grace@example.com", "ue2")
	if err != nil || exists {
		t.Fatalf("self-edit should not report conflict: err=%v exists=%v", err, exists)
	}

	// Another user with the same username.
	u2 := &model.User{ID: "ue3", Username: "heidi", Email: "heidi@example.com", PasswordHash: "h", Role: "user"}
	r.Create(ctx, u2)
	exists, _ = r.ExistsByUsernameOrEmailExcept(ctx, "grace", "unique@example.com", "ue3")
	if !exists {
		t.Fatal("expected conflict with existing username")
	}
}

func TestUserRepo_Count(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	n, err := r.Count(ctx)
	if err != nil || n != 0 {
		t.Fatalf("empty DB count: %v %d", err, n)
	}

	r.Create(ctx, &model.User{ID: "uc1", Username: "ivan", Email: "ivan@example.com", PasswordHash: "h", Role: "user"})

	n, _ = r.Count(ctx)
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}

func TestUserRepo_TryConsumeStorage_WithinQuota(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "q1", Username: "q1", Email: "q1@x", PasswordHash: "h", StorageLimit: 1000, StorageUsed: 900}
	if err := r.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	ok, err := r.TryConsumeStorage(ctx, "q1", 100)
	if err != nil || !ok {
		t.Fatalf("consume 100 of 100 free: ok=%v err=%v", ok, err)
	}

	got, _ := r.GetByID(ctx, "q1")
	if got.StorageUsed != 1000 {
		t.Fatalf("expected storage_used=1000, got %d", got.StorageUsed)
	}
}

func TestUserRepo_TryConsumeStorage_ExactBoundary(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "q2", Username: "q2", Email: "q2@x", PasswordHash: "h", StorageLimit: 1000, StorageUsed: 0}
	r.Create(ctx, u)

	ok, err := r.TryConsumeStorage(ctx, "q2", 1000)
	if err != nil || !ok {
		t.Fatalf("exactly filling quota should succeed: ok=%v err=%v", ok, err)
	}

	ok, err = r.TryConsumeStorage(ctx, "q2", 1)
	if err != nil || ok {
		t.Fatalf("one byte past full should fail: ok=%v err=%v", ok, err)
	}

	got, _ := r.GetByID(ctx, "q2")
	if got.StorageUsed != 1000 {
		t.Fatalf("quota should not have advanced past limit: got %d", got.StorageUsed)
	}
}

func TestUserRepo_TryConsumeStorage_OverQuota(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "q3", Username: "q3", Email: "q3@x", PasswordHash: "h", StorageLimit: 1000, StorageUsed: 900}
	r.Create(ctx, u)

	ok, err := r.TryConsumeStorage(ctx, "q3", 101)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("consuming 101 when 100 free should fail")
	}

	got, _ := r.GetByID(ctx, "q3")
	if got.StorageUsed != 900 {
		t.Fatalf("failed consume must not mutate counter: got %d", got.StorageUsed)
	}
}

func TestUserRepo_TryConsumeStorage_UnlimitedBypassesLimit(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "q4", Username: "q4", Email: "q4@x", PasswordHash: "h", StorageLimit: -1, StorageUsed: 0}
	r.Create(ctx, u)

	// A ludicrously large delta that would overflow any normal quota.
	const huge = int64(1 << 60)
	ok, err := r.TryConsumeStorage(ctx, "q4", huge)
	if err != nil || !ok {
		t.Fatalf("unlimited user should accept any positive delta: ok=%v err=%v", ok, err)
	}

	got, _ := r.GetByID(ctx, "q4")
	if got.StorageUsed != huge {
		t.Fatalf("storage_used should reflect the delta even on unlimited: got %d", got.StorageUsed)
	}
}

func TestUserRepo_TryConsumeStorage_MissingUser(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	ok, err := r.TryConsumeStorage(ctx, "ghost", 1)
	if err != nil {
		t.Fatalf("missing user should not error: %v", err)
	}
	if ok {
		t.Fatal("missing user must not be reported as consumed")
	}
}

func TestUserRepo_TryConsumeStorage_RejectsNonPositiveDelta(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "q5", Username: "q5", Email: "q5@x", PasswordHash: "h", StorageLimit: 1000}
	r.Create(ctx, u)

	for _, d := range []int64{0, -1, -1024} {
		if ok, err := r.TryConsumeStorage(ctx, "q5", d); err == nil || ok {
			t.Fatalf("delta=%d: expected error, got ok=%v err=%v", d, ok, err)
		}
	}
}

// TestUserRepo_TryConsumeStorage_ConcurrentConsume verifies that when N
// goroutines race to consume storage that only fits M<N of them, exactly M
// succeed and the counter ends up at exactly the limit. This is the core
// TOCTOU regression test: the old check-then-update pattern allowed all N to
// pass the check and bump the counter past the limit.
func TestUserRepo_TryConsumeStorage_ConcurrentConsume(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	const (
		limit        = int64(10_000)
		deltaPerCall = int64(1_000)
		goroutines   = 20 // only 10 should succeed; the other 10 should fail
		expectedOK   = int32(10)
	)

	u := &model.User{ID: "qc", Username: "qc", Email: "qc@x", PasswordHash: "h", StorageLimit: limit, StorageUsed: 0}
	r.Create(ctx, u)

	var successes int32
	var failures int32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start // release all goroutines at once
			ok, err := r.TryConsumeStorage(ctx, "qc", deltaPerCall)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if ok {
				atomic.AddInt32(&successes, 1)
			} else {
				atomic.AddInt32(&failures, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if successes != expectedOK {
		t.Fatalf("expected exactly %d successes, got %d (failures=%d)", expectedOK, successes, failures)
	}
	if failures != int32(goroutines)-expectedOK {
		t.Fatalf("expected %d failures, got %d", goroutines-int(expectedOK), failures)
	}

	got, _ := r.GetByID(ctx, "qc")
	if got.StorageUsed != limit {
		t.Fatalf("counter should match limit exactly; got %d want %d", got.StorageUsed, limit)
	}
}

func TestUserRepo_ReleaseStorage(t *testing.T) {
	r := NewUserRepo(newTestDB(t))
	ctx := context.Background()

	u := &model.User{ID: "r1", Username: "r1", Email: "r1@x", PasswordHash: "h", StorageLimit: 1000, StorageUsed: 500}
	r.Create(ctx, u)

	if err := r.ReleaseStorage(ctx, "r1", 300); err != nil {
		t.Fatalf("ReleaseStorage: %v", err)
	}
	got, _ := r.GetByID(ctx, "r1")
	if got.StorageUsed != 200 {
		t.Fatalf("expected 200, got %d", got.StorageUsed)
	}

	if err := r.ReleaseStorage(ctx, "r1", 0); err == nil {
		t.Fatal("release of 0 should error")
	}
	if err := r.ReleaseStorage(ctx, "r1", -10); err == nil {
		t.Fatal("release of negative should error")
	}
}

func TestUser_HasStorageSpace(t *testing.T) {
	u := &model.User{StorageLimit: 1000, StorageUsed: 900}
	if !u.HasStorageSpace(100) {
		t.Fatal("should have space for exactly 100 bytes")
	}
	if u.HasStorageSpace(101) {
		t.Fatal("should not have space for 101 bytes")
	}

	u.StorageLimit = -1
	if !u.HasStorageSpace(1 << 62) {
		t.Fatal("unlimited storage should always have space")
	}
}
