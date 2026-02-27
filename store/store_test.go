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
	t.Setenv("HOME", t.TempDir())
	db, err := Open()
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func TestExtractTagsFromText(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"fix the #auth login bug", []string{"#auth"}},
		{"#todo and also #refactor this", []string{"#todo", "#refactor"}},
		{"no tags here", nil},
		{"repeated #todo #todo tag", []string{"#todo", "#todo"}}, // dedup is caller's responsibility
	}
	for _, c := range cases {
		got := ExtractTagsFromText(c.input)
		if len(got) != len(c.expected) {
			t.Errorf("input %q: expected %v, got %v", c.input, c.expected, got)
			continue
		}
		for i := range got {
			if got[i] != c.expected[i] {
				t.Errorf("input %q: expected %v at index %d, got %v", c.input, c.expected[i], i, got[i])
			}
		}
	}
}

func TestSaveNote(t *testing.T) {
	t.Run("user tags stored in note_tags", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		if err := SaveNote(db, "hello world", "#todo"); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		var tag string
		row := db.QueryRow(`SELECT tag FROM note_tags WHERE note_id = 1`)
		if err := row.Scan(&tag); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if tag != "#todo" {
			t.Errorf("expected tag %q, got %q", "#todo", tag)
		}
	})

	t.Run("tags extracted from note body", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		if err := SaveNote(db, "fix the #auth login bug", ""); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		var tag string
		row := db.QueryRow(`SELECT tag FROM note_tags WHERE note_id = 1`)
		if err := row.Scan(&tag); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if tag != "#auth" {
			t.Errorf("expected extracted tag %q, got %q", "#auth", tag)
		}
	})

	t.Run("body and user tags are merged and deduplicated", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		// #auth appears in both body and user tags
		if err := SaveNote(db, "fix the #auth login bug", "#auth #bug"); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		rows, err := db.Query(`SELECT tag FROM note_tags WHERE note_id = 1 ORDER BY tag`)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()

		var tags []string
		for rows.Next() {
			var tag string
			rows.Scan(&tag)
			tags = append(tags, tag)
		}

		if len(tags) != 2 {
			t.Errorf("expected 2 distinct tags, got %d: %v", len(tags), tags)
		}
	})
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

	t.Run("Tags field is populated", func(t *testing.T) {
		notes, err := QueryNotes(db, []string{"#todo"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notes) != 1 {
			t.Fatalf("expected 1 note, got %d", len(notes))
		}
		if len(notes[0].Tags) == 0 {
			t.Errorf("expected Tags to be populated, got empty slice")
		}
	})
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
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

func TestMigrationFromOldSchema(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Simulate an existing DB at schema version 1 (notes table with tags column).
	dbPath := filepath.Join(dir, ".seton", "notes.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	seed, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	seed.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL)`)
	seed.Exec(`INSERT INTO schema_version VALUES (1)`)
	seed.Exec(`CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, text TEXT NOT NULL, tags TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`)
	seed.Exec(`INSERT INTO notes (text, tags) VALUES ('old note', '#legacy #test')`)
	seed.Close()

	// Open through normal path — should run migrations 2, 3, 4.
	db, err := Open()
	if err != nil {
		t.Fatalf("Open after migration: %v", err)
	}
	defer db.Close()

	// Legacy tags should have been migrated to note_tags.
	rows, err := db.Query(`SELECT tag FROM note_tags ORDER BY tag`)
	if err != nil {
		t.Fatalf("query note_tags: %v", err)
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var tag string
		rows.Scan(&tag)
		tags = append(tags, tag)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 migrated tags, got %d: %v", len(tags), tags)
	}
}
