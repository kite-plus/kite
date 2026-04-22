package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/amigoer/kite/internal/model"
)

// TestFileService_UploadBumpsStorageUsed verifies the happy path: a successful
// upload increases the user's storage_used counter by exactly the byte count,
// with no double-accounting (earlier versions of Upload both reserved quota
// and ran a separate post-write UpdateStorageUsed).
func TestFileService_UploadBumpsStorageUsed(t *testing.T) {
	svc, cleanup, _ := newUploadPathTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Baseline: the helper provisions u1 with StorageLimit=-1 and StorageUsed=0.
	before, err := fetchUser(ctx, svc, "u1")
	if err != nil {
		t.Fatalf("fetch baseline: %v", err)
	}
	if before.StorageUsed != 0 {
		t.Fatalf("expected baseline storage_used=0, got %d", before.StorageUsed)
	}

	body := bytes.Repeat([]byte("q"), 4096)
	result, err := svc.Upload(ctx, UploadParams{
		UserID:   "u1",
		Filename: "quota.txt",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.File.SizeBytes != int64(len(body)) {
		t.Fatalf("file.SizeBytes = %d, want %d", result.File.SizeBytes, len(body))
	}

	after, err := fetchUser(ctx, svc, "u1")
	if err != nil {
		t.Fatalf("fetch after: %v", err)
	}
	if after.StorageUsed != int64(len(body)) {
		t.Fatalf("storage_used = %d, want %d (no double-accounting)", after.StorageUsed, len(body))
	}
}

// TestFileService_UploadRejectsOverQuotaAtomically verifies that when the body
// exceeds the remaining quota at write time, the atomic reservation refuses
// the upload and the counter is not mutated.
func TestFileService_UploadRejectsOverQuotaAtomically(t *testing.T) {
	svc, cleanup, _ := newUploadPathTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Tighten u1 to a small quota.
	u, err := svc.userRepo.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("get u1: %v", err)
	}
	u.StorageLimit = 100
	u.StorageUsed = 90
	if err := svc.userRepo.Update(ctx, u); err != nil {
		t.Fatalf("update u1: %v", err)
	}

	// Body is larger than the remaining 10 bytes.
	body := bytes.Repeat([]byte("z"), 50)
	_, err = svc.Upload(ctx, UploadParams{
		UserID:   "u1",
		Filename: "big.txt",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if !errors.Is(err, ErrStorageFull) {
		t.Fatalf("expected ErrStorageFull, got %v", err)
	}

	after, err := svc.userRepo.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("fetch after: %v", err)
	}
	if after.StorageUsed != 90 {
		t.Fatalf("failed upload must not mutate counter; got %d want 90", after.StorageUsed)
	}
}

// TestFileService_UploadGuestBypassesQuotaCounter verifies that guest uploads
// don't touch any user's storage counter.
func TestFileService_UploadGuestBypassesQuotaCounter(t *testing.T) {
	svc, cleanup, _ := newUploadPathTestService(t)
	defer cleanup()

	ctx := context.Background()

	before, err := svc.userRepo.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("fetch before: %v", err)
	}
	initial := before.StorageUsed

	body := bytes.Repeat([]byte("g"), 2048)
	_, err = svc.Upload(ctx, UploadParams{
		UserID:   "guest",
		IsGuest:  true,
		Filename: "anon.txt",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("guest Upload: %v", err)
	}

	after, err := svc.userRepo.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("fetch after: %v", err)
	}
	if after.StorageUsed != initial {
		t.Fatalf("guest upload must not bump other users' counters; got %d want %d", after.StorageUsed, initial)
	}
}

// TestFileService_UploadDedupDoesNotConsumeQuota verifies that a duplicate
// upload returns the existing file without touching the counter — important
// because the old code would pre-check before the dedup match.
func TestFileService_UploadDedupDoesNotConsumeQuota(t *testing.T) {
	svc, cleanup, _ := newUploadPathTestService(t)
	defer cleanup()

	ctx := context.Background()

	body := bytes.Repeat([]byte("d"), 1024)
	first, err := svc.Upload(ctx, UploadParams{
		UserID:   "u1",
		Filename: "dup.txt",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("first Upload: %v", err)
	}

	afterFirst, _ := svc.userRepo.GetByID(ctx, "u1")
	firstUsed := afterFirst.StorageUsed
	if firstUsed != int64(len(body)) {
		t.Fatalf("first upload should have bumped counter to %d, got %d", len(body), firstUsed)
	}

	second, err := svc.Upload(ctx, UploadParams{
		UserID:   "u1",
		Filename: "dup.txt",
		Reader:   bytes.NewReader(body),
		Size:     int64(len(body)),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("second Upload: %v", err)
	}
	if second.File.ID != first.File.ID {
		t.Fatalf("dedup should return the same file id; got %q want %q", second.File.ID, first.File.ID)
	}

	afterSecond, _ := svc.userRepo.GetByID(ctx, "u1")
	if afterSecond.StorageUsed != firstUsed {
		t.Fatalf("dedup upload must not bump counter; got %d want %d", afterSecond.StorageUsed, firstUsed)
	}
}

// fetchUser is a tiny helper so the tests don't need to reach into the repo.
func fetchUser(ctx context.Context, svc *FileService, id string) (*model.User, error) {
	return svc.userRepo.GetByID(ctx, id)
}
