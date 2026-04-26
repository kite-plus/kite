package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/kite-plus/kite/internal/i18n"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

// errInstalled is returned when a database-mutating setup endpoint is hit
// after a user has already been created. Lets callers surface a friendly
// 400 instead of leaking a stack trace.
var errInstalled = errors.New("system is already initialised")

// SetupDatabaseHandler wires the database-stage of the install wizard. It is
// kept separate from [SetupHandler] because that one needs a gorm.DB pointing
// at the *current* database, while this handler is invoked *before* the
// operator has decided which database to use — and so it has to be able to
// open arbitrary connections at runtime to validate them.
type SetupDatabaseHandler struct {
	userRepoCount   func() (int64, error) // probe — has anyone been created yet?
	persistDatabase func(driver, dsn string) error
}

// NewSetupDatabaseHandler builds the handler. The two callbacks wire the
// handler to the rest of the application:
//
//   - userRepoCount returns the row count of the user table; the handler uses
//     it as the "are we already installed?" gate so a malicious caller can't
//     overwrite the running config after install.
//   - persistDatabase writes the chosen driver / DSN to the on-disk config
//     file so it survives a restart. main.go owns the actual file path.
func NewSetupDatabaseHandler(
	userRepoCount func() (int64, error),
	persistDatabase func(driver, dsn string) error,
) *SetupDatabaseHandler {
	return &SetupDatabaseHandler{
		userRepoCount:   userRepoCount,
		persistDatabase: persistDatabase,
	}
}

// databaseChoiceRequest is the payload accepted by both /setup/test-database
// and /setup/database. The wizard's UI assembles the DSN client-side before
// posting it; we don't accept structured host/user/pass fields because the
// permutations across mysql/postgres/sqlite would explode the validation
// matrix and operators can paste the exact DSN they want.
type databaseChoiceRequest struct {
	Driver string `json:"driver" binding:"required"`
	DSN    string `json:"dsn" binding:"required"`
}

// TestDatabase opens a short-lived gorm connection using the supplied
// driver/DSN, runs a `SELECT 1` to prove the credentials work, and then
// closes the pool. The endpoint exists so the wizard can surface "wrong
// password" / "host unreachable" errors *before* the operator commits the
// choice and triggers a restart — fixing a typo after the restart would mean
// the server boots into a broken state and the wizard becomes unreachable.
func (h *SetupDatabaseHandler) TestDatabase(c *gin.Context) {
	if h.installedGuard(c) {
		return
	}
	var req databaseChoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeySetupInvalidDatabase, err.Error()))
		return
	}
	if err := probeDatabase(req.Driver, req.DSN); err != nil {
		Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	Success(c, gin.H{"ok": true, "message": "connection ok"})
}

// SaveDatabase persists the supplied driver / DSN to the config file so the
// next process boot uses it. The endpoint deliberately runs the same probe
// as [TestDatabase] before writing — without this guard a typo would bake a
// broken config into disk and the next boot would fail before the wizard
// can render again.
//
// On success the response carries `restart_required: true` so the SPA / page
// can render an explicit "please restart" panel; we don't try to self-restart
// because some deployments are not under a supervisor and would fail to come
// back up.
func (h *SetupDatabaseHandler) SaveDatabase(c *gin.Context) {
	if h.installedGuard(c) {
		return
	}
	var req databaseChoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeySetupInvalidDatabase, err.Error()))
		return
	}
	if err := probeDatabase(req.Driver, req.DSN); err != nil {
		Fail(c, http.StatusBadRequest, 40000, err.Error())
		return
	}
	if err := h.persistDatabase(req.Driver, req.DSN); err != nil {
		ServerError(c, M(c, i18n.KeySetupSaveDatabaseFailed, err.Error()))
		return
	}
	Success(c, gin.H{
		"restart_required": true,
		"message":          "database config saved; please restart the service",
	})
}

// installedGuard returns true (and writes a 400) when the system is already
// installed. Both database endpoints are expected to be no-ops once a user
// row exists; otherwise an attacker who slipped past auth could trash the
// running config.
func (h *SetupDatabaseHandler) installedGuard(c *gin.Context) bool {
	count, err := h.userRepoCount()
	if err == nil && count > 0 {
		BadRequest(c, errInstalled.Error())
		return true
	}
	return false
}

// probeDatabase opens a fresh gorm connection, runs `SELECT 1` and closes the
// pool. The 5-second timeout prevents a stalled DNS lookup or unreachable
// host from hanging the request indefinitely; gorm itself doesn't surface a
// connect-timeout knob through its dialectors.
func probeDatabase(driver, dsn string) error {
	driver = strings.TrimSpace(strings.ToLower(driver))
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return errors.New("DSN is empty")
	}

	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return fmt.Errorf("unsupported driver %q (expected sqlite, mysql or postgres)", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlog.Default.LogMode(gormlog.Silent),
	})
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("acquire pool: %w", err)
	}
	defer sqlDB.Close()

	// SELECT 1 is the universal "is this database alive?" probe — it works
	// across sqlite/mysql/postgres without needing a specific table. The
	// 5-second cap prevents a stalled DNS lookup or unreachable host from
	// hanging the request indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return nil
}
