package appinfra

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOpenDBFromEnv_SQLite verifies that when DB_URI uses the sqlite scheme,
// the bootstrap opens a local SQLite database and basic SQL works.
func TestOpenDBFromEnv_SQLite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "app.db")
	os.Setenv("DB_URI", "sqlite://"+dbFile)
	os.Setenv("LITE_MODE", "1")
	t.Cleanup(func() { os.Unsetenv("DB_URI") })

	db, err := openDBFromEnv()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Simple schema and CRUD to confirm the connection is usable.
	err = db.Exec(`
        CREATE TABLE IF NOT EXISTS t (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL
        );
    `).Error
	assert.NoError(t, err)

	err = db.Exec("INSERT INTO t(name) VALUES (?), (?)", "a", "b").Error
	assert.NoError(t, err)

	var cnt int
	err = db.Raw("SELECT COUNT(1) FROM t").Scan(&cnt).Error
	assert.NoError(t, err)
	assert.Equal(t, 2, cnt)
}
