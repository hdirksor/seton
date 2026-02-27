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

func TestQueryNotes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	SaveNote(db, "fix the login bug", "#bug #auth")
	SaveNote(db, "refactor user service", "#refactor #auth")
	SaveNote(db, "add dark mode", "#todo")

	t.Run("single tag match", func(t *testing.T) {
		notes, err := QueryNotes(db, []string{"#auth"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 2 {
			t.Errorf("expected 2 notes, got %d", len(notes))
		}
	})

	t.Run("AND match", func(t *testing.T) {
		notes, err := QueryNotes(db, []string{"#auth", "#bug"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 1 {
			t.Errorf("expected 1 note, got %d", len(notes))
		}
		if notes[0].Text != "fix the login bug" {
			t.Errorf("unexpected note text: %q", notes[0].Text)
		}
	})

	t.Run("no match", func(t *testing.T) {
		notes, err := QueryNotes(db, []string{"#missing"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 0 {
			t.Errorf("expected 0 notes, got %d", len(notes))
		}
	})

	t.Run("no tags returns all", func(t *testing.T) {
		notes, err := QueryNotes(db, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 3 {
			t.Errorf("expected 3 notes, got %d", len(notes))
		}
	})

	t.Run("no false prefix match", func(t *testing.T) {
		// #auth should not match a hypothetical #auth-admin tag
		SaveNote(db, "edge case note", "#auth-admin")
		notes, err := QueryNotes(db, []string{"#auth"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 2 {
			t.Errorf("expected 2 notes (not the #auth-admin one), got %d", len(notes))
		}
	})
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
