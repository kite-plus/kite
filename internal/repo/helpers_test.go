package repo

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/amigoer/kite/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// dbCounter ensures every in-memory database has a unique name even when tests
// run concurrently.  SQLite in-memory databases with the same name share state
// within the same process, which causes cross-test contamination.
var dbCounter int64

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	id := atomic.AddInt64(&dbCounter, 1)
	dsn := fmt.Sprintf("file:repo-test-%d?mode=memory&cache=shared", id)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserIdentity{},
		&model.Album{},
		&model.APIToken{},
		&model.File{},
		&model.Setting{},
		&model.StorageConfig{},
		&model.FileAccessLog{},
		&model.FileReplica{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}
