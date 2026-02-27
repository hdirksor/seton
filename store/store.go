package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "modernc.org/sqlite"
)

var tagRegexp = regexp.MustCompile(`#[a-zA-Z0-9_-]+`)

// migration holds a SQL statement and an optional Go function to run in the same transaction.
type migration struct {
	sql string
	fn  func(*sql.Tx) error
}

var migrations = []migration{
	// 1: initial notes table (with legacy tags column)
	{sql: `CREATE TABLE IF NOT EXISTS notes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		text       TEXT NOT NULL,
		tags       TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`},
	// 2: separate tags table
	{sql: `CREATE TABLE IF NOT EXISTS note_tags (
		note_id INTEGER NOT NULL REFERENCES notes(id),
		tag     TEXT NOT NULL,
		UNIQUE(note_id, tag)
	)`},
	// 3: migrate space-separated tags from notes.tags into note_tags
	{fn: migrateTagsToTable},
	// 4: drop the now-redundant tags column
	{sql: `ALTER TABLE notes DROP COLUMN tags`},
}

func migrateTagsToTable(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id, tags FROM notes WHERE tags IS NOT NULL AND tags != ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type record struct {
		id   int64
		tags string
	}
	var records []record
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.id, &r.tags); err != nil {
			return err
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	for _, r := range records {
		for _, tag := range strings.Fields(r.tags) {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO note_tags (note_id, tag) VALUES (?, ?)`, r.id, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for i, m := range migrations {
		version := i + 1
		if version <= current {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migration %d begin: %w", version, err)
		}
		if m.sql != "" {
			if _, err := tx.Exec(m.sql); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d: %w", version, err)
			}
		}
		if m.fn != nil {
			if err := m.fn(tx); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d: %w", version, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d record version: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d commit: %w", version, err)
		}
	}
	return nil
}

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
	if err := applyMigrations(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// ExtractTagsFromText returns all #tag tokens found in text.
func ExtractTagsFromText(text string) []string {
	return tagRegexp.FindAllString(text, -1)
}

// mergeTags combines body-extracted tags and user-provided tags, deduplicating
// and normalizing (adding # prefix where missing).
func mergeTags(bodyTags, userTags []string) []string {
	seen := map[string]bool{}
	var out []string

	for _, tag := range bodyTags {
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	for _, tag := range userTags {
		if !strings.HasPrefix(tag, "#") {
			tag = "#" + tag
		}
		if !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out
}

func SaveNote(db *sql.DB, text, userTags string) error {
	tags := mergeTags(ExtractTagsFromText(text), strings.Fields(userTags))

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	result, err := tx.Exec(`INSERT INTO notes (text) VALUES (?)`, text)
	if err != nil {
		tx.Rollback()
		return err
	}

	noteID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, tag := range tags {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO note_tags (note_id, tag) VALUES (?, ?)`, noteID, tag); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// Note represents a row from the notes table with its associated tags.
type Note struct {
	ID        int
	Text      string
	Tags      []string
	CreatedAt string
}

// QueryNotes returns all notes containing every provided tag (AND logic).
// If tags is empty, all notes are returned.
func QueryNotes(db *sql.DB, tags []string) ([]Note, error) {
	query := `SELECT n.id, n.text, n.created_at, GROUP_CONCAT(nt.tag, ' ')
	          FROM notes n
	          LEFT JOIN note_tags nt ON nt.note_id = n.id`

	var args []any

	if len(tags) > 0 {
		subqueries := make([]string, len(tags))
		for i, tag := range tags {
			subqueries[i] = `SELECT note_id FROM note_tags WHERE tag = ?`
			args = append(args, tag)
		}
		query += ` WHERE n.id IN (` + strings.Join(subqueries, ` INTERSECT `) + `)`
	}

	query += ` GROUP BY n.id ORDER BY n.created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		var tagsStr sql.NullString
		if err := rows.Scan(&n.ID, &n.Text, &n.CreatedAt, &tagsStr); err != nil {
			return nil, err
		}
		if tagsStr.Valid && tagsStr.String != "" {
			n.Tags = strings.Fields(tagsStr.String)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}
