package sqlite

import (
	"fmt"
	"os"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// New opens a SQLite database using DSN from environment variable SQLITE_DSN.
// Fallback: file:./var/data/sqlite/app.db
func New() (*gorm.DB, error) {
	dsn := os.Getenv("SQLITE_DSN")
	if dsn == "" {
		dsn = "file:./var/data/sqlite/app.db?mode=rwc&cache=shared"
	}
	return NewWithDSN(dsn)
}

// NewWithDSN opens a SQLite database with the provided DSN and applies
// sensible pragmas for a single-binary embedded setup.
func NewWithDSN(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(gormsqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("sqlite open, dsn: %s, err: %w", dsn, err)
	}

	// Apply recommended pragmas. Errors are returned to surface misconfigurations.
	if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
		return nil, fmt.Errorf("apply PRAGMA journal_mode=WAL: %w", err)
	}
	if err := db.Exec("PRAGMA synchronous=NORMAL;").Error; err != nil {
		return nil, fmt.Errorf("apply PRAGMA synchronous=NORMAL: %w", err)
	}
	if err := db.Exec("PRAGMA foreign_keys=ON;").Error; err != nil {
		return nil, fmt.Errorf("apply PRAGMA foreign_keys=ON: %w", err)
	}
	// Keep cache in memory for speed; negative value indicates KiB
	if err := db.Exec("PRAGMA cache_size=-20000;").Error; err != nil {
		return nil, fmt.Errorf("apply PRAGMA cache_size: %w", err)
	}
	if err := db.Exec("PRAGMA temp_store=MEMORY;").Error; err != nil {
		return nil, fmt.Errorf("apply PRAGMA temp_store: %w", err)
	}

	return db, nil
}
