package repo

import (
	"context"
	"fmt"
	"testing"

	"github.com/amigoer/kite/internal/model"
)

func TestAlbumRepo_CreateAndGetByID(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	a := &model.Album{ID: "al1", UserID: "u1", Name: "Root Folder"}
	if err := r.Create(ctx, a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := r.GetByID(ctx, "al1")
	if err != nil || got.Name != "Root Folder" {
		t.Fatalf("GetByID: err=%v got=%v", err, got)
	}
}

func TestAlbumRepo_GetByID_NotFound(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	_, err := r.GetByID(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for missing album")
	}
}

func TestAlbumRepo_Update(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	a := &model.Album{ID: "al2", UserID: "u1", Name: "Old Name"}
	r.Create(ctx, a)

	a.Name = "New Name"
	if err := r.Update(ctx, a); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := r.GetByID(ctx, "al2")
	if got.Name != "New Name" {
		t.Fatalf("name not updated: %s", got.Name)
	}
}

func TestAlbumRepo_Delete_UnlinksFiles(t *testing.T) {
	db := newTestDB(t)
	r := NewAlbumRepo(db)
	fr := NewFileRepo(db)
	ctx := context.Background()

	a := &model.Album{ID: "al3", UserID: "u1", Name: "Folder"}
	r.Create(ctx, a)

	albumID := "al3"
	f := &model.File{
		ID: "f1", UserID: "u1", AlbumID: &albumID,
		StorageConfigID: "s1", OriginalName: "pic.jpg", StorageKey: "k1",
		HashMD5: "aabbccdd" + "aabbccddaabbccdd" + "aabbccdd", SizeBytes: 100,
		MimeType: "image/jpeg", FileType: model.FileTypeImage, URL: "/f/1",
	}
	fr.Create(ctx, f)

	if err := r.Delete(ctx, "al3"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Album row should be gone.
	if _, err := r.GetByID(ctx, "al3"); err == nil {
		t.Fatal("album should be deleted")
	}

	// File should still exist but with cleared album_id.
	gotFile, err := fr.GetByID(ctx, "f1")
	if err != nil {
		t.Fatalf("file should still exist: %v", err)
	}
	if gotFile.AlbumID != nil {
		t.Fatalf("album_id should be cleared, got %v", gotFile.AlbumID)
	}
}

func TestAlbumRepo_Delete_Recursive(t *testing.T) {
	db := newTestDB(t)
	r := NewAlbumRepo(db)
	ctx := context.Background()

	// Build a 3-level hierarchy: root → child → grandchild.
	root := &model.Album{ID: "root", UserID: "u1", Name: "root"}
	r.Create(ctx, root)

	childParent := "root"
	child := &model.Album{ID: "child", UserID: "u1", Name: "child", ParentID: &childParent}
	r.Create(ctx, child)

	gcParent := "child"
	grandchild := &model.Album{ID: "gc", UserID: "u1", Name: "grandchild", ParentID: &gcParent}
	r.Create(ctx, grandchild)

	if err := r.Delete(ctx, "root"); err != nil {
		t.Fatalf("recursive Delete: %v", err)
	}

	for _, id := range []string{"root", "child", "gc"} {
		if _, err := r.GetByID(ctx, id); err == nil {
			t.Fatalf("album %s should be deleted", id)
		}
	}
}

func TestAlbumRepo_ListByUser_RootOnly(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.Album{ID: "ra1", UserID: "u1", Name: "A"})
	r.Create(ctx, &model.Album{ID: "ra2", UserID: "u1", Name: "B"})
	parent := "ra1"
	r.Create(ctx, &model.Album{ID: "ra3", UserID: "u1", Name: "Child", ParentID: &parent})

	albums, total, err := r.ListByUser(ctx, "u1", nil, 1, 10)
	if err != nil || total != 2 || len(albums) != 2 {
		t.Fatalf("ListByUser root: err=%v total=%d len=%d", err, total, len(albums))
	}
}

func TestAlbumRepo_ListByUser_WithParent(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.Album{ID: "pb1", UserID: "u1", Name: "Parent"})
	p := "pb1"
	r.Create(ctx, &model.Album{ID: "pb2", UserID: "u1", Name: "Child1", ParentID: &p})
	r.Create(ctx, &model.Album{ID: "pb3", UserID: "u1", Name: "Child2", ParentID: &p})

	albums, total, err := r.ListByUser(ctx, "u1", &p, 1, 10)
	if err != nil || total != 2 || len(albums) != 2 {
		t.Fatalf("ListByUser with parent: err=%v total=%d len=%d", err, total, len(albums))
	}
}

func TestAlbumRepo_ListByUser_Pagination(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		r.Create(ctx, &model.Album{ID: fmt.Sprintf("pg%d", i), UserID: "u1", Name: fmt.Sprintf("Folder%d", i)})
	}

	page1, total, _ := r.ListByUser(ctx, "u1", nil, 1, 3)
	if total != 5 || len(page1) != 3 {
		t.Fatalf("page1: total=%d len=%d", total, len(page1))
	}
	page2, _, _ := r.ListByUser(ctx, "u1", nil, 2, 3)
	if len(page2) != 2 {
		t.Fatalf("page2: len=%d", len(page2))
	}
}

func TestAlbumRepo_CountChildren(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.Album{ID: "cc1", UserID: "u1", Name: "Parent"})
	p := "cc1"
	r.Create(ctx, &model.Album{ID: "cc2", UserID: "u1", Name: "Child1", ParentID: &p})
	r.Create(ctx, &model.Album{ID: "cc3", UserID: "u1", Name: "Child2", ParentID: &p})

	n, err := r.CountChildren(ctx, "cc1")
	if err != nil || n != 2 {
		t.Fatalf("CountChildren: err=%v n=%d", err, n)
	}
}

func TestAlbumRepo_ListAncestors(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.Album{ID: "anc1", UserID: "u1", Name: "L1"})
	p1 := "anc1"
	r.Create(ctx, &model.Album{ID: "anc2", UserID: "u1", Name: "L2", ParentID: &p1})
	p2 := "anc2"
	r.Create(ctx, &model.Album{ID: "anc3", UserID: "u1", Name: "L3", ParentID: &p2})

	ancestors, err := r.ListAncestors(ctx, "u1", "anc3")
	if err != nil {
		t.Fatalf("ListAncestors: %v", err)
	}
	// ListAncestors includes the node itself: anc1 + anc2 + anc3 = 3 entries.
	if len(ancestors) != 3 {
		t.Fatalf("expected 3 entries (node + ancestors), got %d", len(ancestors))
	}
}

func TestAlbumRepo_ListAncestors_WrongUser(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.Album{ID: "aw1", UserID: "u1", Name: "Folder"})
	_, err := r.ListAncestors(ctx, "other-user", "aw1")
	if err == nil {
		t.Fatal("expected error for wrong user")
	}
}

func TestAlbumRepo_ListPublic(t *testing.T) {
	r := NewAlbumRepo(newTestDB(t))
	ctx := context.Background()

	r.Create(ctx, &model.Album{ID: "pub1", UserID: "u1", Name: "Public", IsPublic: true})
	r.Create(ctx, &model.Album{ID: "prv1", UserID: "u1", Name: "Private", IsPublic: false})

	albums, total, err := r.ListPublic(ctx, 1, 10)
	if err != nil || total != 1 || len(albums) != 1 || albums[0].ID != "pub1" {
		t.Fatalf("ListPublic: err=%v total=%d len=%d", err, total, len(albums))
	}
}
