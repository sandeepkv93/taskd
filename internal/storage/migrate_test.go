package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrateRoundTripCompatibility(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate-roundtrip.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := MigrateUp(db); err != nil {
		t.Fatalf("first migrate up failed: %v", err)
	}

	if err := MigrateDown(db); err != nil {
		t.Fatalf("migrate down failed: %v", err)
	}

	if err := MigrateUp(db); err != nil {
		t.Fatalf("second migrate up failed: %v", err)
	}

	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}

	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	if err := repo.CreateTask(t.Context(), Task{
		ID:          "task-rt-1",
		Title:       "Roundtrip task",
		Description: "migration compatibility",
		State:       "Inbox",
		Priority:    "Medium",
		Energy:      "Light",
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("insert after roundtrip failed: %v", err)
	}

	got, err := repo.GetTask(t.Context(), "task-rt-1")
	if err != nil {
		t.Fatalf("get after roundtrip failed: %v", err)
	}
	if got.Title != "Roundtrip task" {
		t.Fatalf("unexpected title after roundtrip: %q", got.Title)
	}
}
