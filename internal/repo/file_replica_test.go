package repo

import (
	"context"
	"testing"

	"github.com/amigoer/kite/internal/model"
)

func TestFileReplicaRepo_CreateAndList(t *testing.T) {
	r := NewFileReplicaRepo(newTestDB(t))
	ctx := context.Background()

	rep := &model.FileReplica{
		ID:              "rep1",
		FileID:          "f1",
		StorageConfigID: "cfg1",
		Status:          model.ReplicaStatusPending,
	}
	if err := r.Create(ctx, rep); err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := r.ListByFile(ctx, "f1")
	if err != nil || len(list) != 1 || list[0].ID != "rep1" {
		t.Fatalf("ListByFile: err=%v len=%d", err, len(list))
	}
}

func TestFileReplicaRepo_ListByFile_Empty(t *testing.T) {
	r := NewFileReplicaRepo(newTestDB(t))
	list, err := r.ListByFile(context.Background(), "nobody")
	if err != nil || len(list) != 0 {
		t.Fatalf("ListByFile empty: err=%v len=%d", err, len(list))
	}
}

func TestFileReplicaRepo_UpdateStatus(t *testing.T) {
	r := NewFileReplicaRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.FileReplica{
		ID: "rep2", FileID: "f1", StorageConfigID: "cfg1",
		Status: model.ReplicaStatusPending,
	})

	if err := r.UpdateStatus(ctx, "rep2", model.ReplicaStatusFailed, "connection timeout"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	list, _ := r.ListByFile(ctx, "f1")
	if list[0].Status != model.ReplicaStatusFailed {
		t.Fatalf("status not updated: %s", list[0].Status)
	}
	if list[0].ErrorMsg != "connection timeout" {
		t.Fatalf("error_msg not updated: %s", list[0].ErrorMsg)
	}
}

func TestFileReplicaRepo_DeleteByFile(t *testing.T) {
	r := NewFileReplicaRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.FileReplica{ID: "rep3", FileID: "fA", StorageConfigID: "cfg1", Status: model.ReplicaStatusOK})
	r.Create(ctx, &model.FileReplica{ID: "rep4", FileID: "fA", StorageConfigID: "cfg2", Status: model.ReplicaStatusOK})
	r.Create(ctx, &model.FileReplica{ID: "rep5", FileID: "fB", StorageConfigID: "cfg1", Status: model.ReplicaStatusOK})

	if err := r.DeleteByFile(ctx, "fA"); err != nil {
		t.Fatalf("DeleteByFile: %v", err)
	}

	listA, _ := r.ListByFile(ctx, "fA")
	if len(listA) != 0 {
		t.Fatalf("fA replicas should be deleted, got %d", len(listA))
	}

	listB, _ := r.ListByFile(ctx, "fB")
	if len(listB) != 1 {
		t.Fatalf("fB replicas should be untouched, got %d", len(listB))
	}
}

func TestFileReplica_StatusConstants(t *testing.T) {
	if model.ReplicaStatusPending != "pending" {
		t.Fatal("wrong pending constant")
	}
	if model.ReplicaStatusOK != "ok" {
		t.Fatal("wrong ok constant")
	}
	if model.ReplicaStatusFailed != "failed" {
		t.Fatal("wrong failed constant")
	}
}
