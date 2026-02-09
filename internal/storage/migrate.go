package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func MigrateUp(db *sql.DB) error {
	return applyMigrations(db, ".up.sql")
}

func MigrateDown(db *sql.DB) error {
	return applyMigrations(db, ".down.sql")
}

func applyMigrations(db *sql.DB, suffix string) error {
	entries, err := fs.Glob(migrationFiles, "migrations/*"+suffix)
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(entries)
	for _, name := range entries {
		sqlBytes, readErr := migrationFiles.ReadFile(name)
		if readErr != nil {
			return fmt.Errorf("read migration %s: %w", name, readErr)
		}
		if _, execErr := db.Exec(string(sqlBytes)); execErr != nil {
			return fmt.Errorf("apply migration %s: %w", name, execErr)
		}
	}
	return nil
}
