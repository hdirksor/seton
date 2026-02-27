package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS notes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		text       TEXT NOT NULL,
		tags       TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestSaveNote(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := SaveNote(db, "hello world", "#todo"); err != nil {
		t.Fatalf("SaveNote returned error: %v", err)
	}

	var text, tags string
	row := db.QueryRow(`SELECT text, tags FROM notes WHERE id = 1`)
	if err := row.Scan(&text, &tags); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if text != "hello world" {
		t.Errorf("expected text %q, got %q", "hello world", text)
	}
	if tags != "#todo" {
		t.Errorf("expected tags %q, got %q", "#todo", tags)
	}
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Override UserHomeDir via HOME env var.
	db, err := Open()
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer db.Close()

	dbPath := filepath.Join(dir, ".seton", "notes.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected db file at %s to exist", dbPath)
	}
}
