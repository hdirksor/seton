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

// tagNamesForNote fetches tag names for a note by joining through the tags table.
func tagNamesForNote(t *testing.T, db *sql.DB, noteID int) []string {
	t.Helper()
	rows, err := db.Query(`
		SELECT t.name FROM note_tags nt
		JOIN tags t ON t.id = nt.tag_id
		WHERE nt.note_id = ?
		ORDER BY t.name`, noteID)
	if err != nil {
		t.Fatalf("tagNamesForNote query: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		names = append(names, name)
	}
	return names
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
	t.Run("user tags stored via tags table", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		if err := SaveNote(db, "hello world", "#todo"); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		names := tagNamesForNote(t, db, 1)
		if len(names) != 1 || names[0] != "#todo" {
			t.Errorf("expected [#todo], got %v", names)
		}
	})

	t.Run("tags extracted from note body", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		if err := SaveNote(db, "fix the #auth login bug", ""); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		names := tagNamesForNote(t, db, 1)
		if len(names) != 1 || names[0] != "#auth" {
			t.Errorf("expected [#auth], got %v", names)
		}
	})

	t.Run("body and user tags are merged and deduplicated", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		if err := SaveNote(db, "fix the #auth login bug", "#auth #bug"); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		names := tagNamesForNote(t, db, 1)
		if len(names) != 2 {
			t.Errorf("expected 2 distinct tags, got %d: %v", len(names), names)
		}
	})

	t.Run("tag row created in tags table", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		if err := SaveNote(db, "a note", "#newtag"); err != nil {
			t.Fatalf("SaveNote returned error: %v", err)
		}

		var name string
		if err := db.QueryRow(`SELECT name FROM tags WHERE name = '#newtag'`).Scan(&name); err != nil {
			t.Errorf("expected tag row in tags table: %v", err)
		}
	})

	t.Run("same tag shared across notes has one row in tags table", func(t *testing.T) {
		db := openTestDB(t)
		defer db.Close()

		SaveNote(db, "first note", "#shared")
		SaveNote(db, "second note", "#shared")

		var count int
		db.QueryRow(`SELECT COUNT(*) FROM tags WHERE name = '#shared'`).Scan(&count)
		if count != 1 {
			t.Errorf("expected 1 row in tags for #shared, got %d", count)
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

	db, err := Open()
	if err != nil {
		t.Fatalf("Open after migration: %v", err)
	}
	defer db.Close()

	// Tags should exist in the tags table.
	rows, err := db.Query(`SELECT name FROM tags ORDER BY name`)
	if err != nil {
		t.Fatalf("query tags: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		names = append(names, name)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 tags in tags table, got %d: %v", len(names), names)
	}

	// note_tags should link the note to both tags via tag_id.
	noteNames := tagNamesForNote(t, db, 1)
	if len(noteNames) != 2 {
		t.Errorf("expected 2 note_tags rows for old note, got %d: %v", len(noteNames), noteNames)
	}
}
