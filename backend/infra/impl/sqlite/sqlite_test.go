package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSQLiteProvider_NewAndCRUD validates basic open + CRUD using the SQLite provider.
// It intentionally uses a file-backed database to exercise WAL and durability pragmas.
func TestSQLiteProvider_NewAndCRUD(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	// rwc ensures DB is created if not exists; cache=shared allows multiple connections if used in future tests.
	dsn := fmt.Sprintf("file:%s?mode=rwc&cache=shared", dbPath)

	db, err := NewWithDSN(dsn)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Ensure WAL mode and some sane pragmas can be applied without error
	// (provider should already apply these, but double-apply is harmless in SQLite)
	assert.NoError(t, db.Exec("PRAGMA journal_mode=WAL;").Error)
	assert.NoError(t, db.Exec("PRAGMA synchronous=NORMAL;").Error)
	assert.NoError(t, db.Exec("PRAGMA foreign_keys=ON;").Error)

	// Basic schema + CRUD
	err = db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            age INTEGER,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    `).Error
	assert.NoError(t, err)

	// Insert
	err = db.Exec("INSERT INTO users (name, age) VALUES (?, ?), (?, ?)", "Alice", 30, "Bob", 25).Error
	assert.NoError(t, err)

	// Update
	err = db.Exec("UPDATE users SET age = ? WHERE name = ?", 31, "Alice").Error
	assert.NoError(t, err)

	// Select
	type row struct {
		Name string
		Age  int
	}
	var r row
	err = db.Raw("SELECT name, age FROM users WHERE name = ?", "Alice").Scan(&r).Error
	assert.NoError(t, err)
	assert.Equal(t, "Alice", r.Name)
	assert.Equal(t, 31, r.Age)

	// Delete
	err = db.Exec("DELETE FROM users WHERE name = ?", "Bob").Error
	assert.NoError(t, err)

	var cnt int
	err = db.Raw("SELECT COUNT(1) FROM users").Scan(&cnt).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, cnt)

	// DB file should exist on disk
	_, statErr := os.Stat(dbPath)
	assert.NoError(t, statErr)
}
