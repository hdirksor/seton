package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open() (*sql.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".seton")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "notes.db"))
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS notes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		text       TEXT NOT NULL,
		tags       TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func SaveNote(db *sql.DB, text, tags string) error {
	_, err := db.Exec(`INSERT INTO notes (text, tags) VALUES (?, ?)`, text, tags)
	return err
}

// Note represents a row from the notes table.
type Note struct {
	ID        int
	Text      string
	Tags      string
	CreatedAt string
}

// QueryNotes returns all notes that contain every tag in the provided list (AND logic).
// If tags is empty, all notes are returned.
func QueryNotes(db *sql.DB, tags []string) ([]Note, error) {
	query := `SELECT id, text, tags, created_at FROM notes`
	args := make([]any, 0, len(tags))

	if len(tags) > 0 {
		query += ` WHERE`
		for i, tag := range tags {
			if i > 0 {
				query += ` AND`
			}
			query += ` (' ' || tags || ' ') LIKE ?`
			args = append(args, "% "+tag+" %")
		}
	}

	query += ` ORDER BY created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Text, &n.Tags, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
