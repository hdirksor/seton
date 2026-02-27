package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdicksonjr/seton/store"
)

func TestBuildFilename(t *testing.T) {
	t.Run("strips hash and joins tags", func(t *testing.T) {
		name := buildFilename([]string{"#todo", "#auth"}, "2026-02-27T14-30-00")
		if !strings.HasPrefix(name, "todo_auth_") {
			t.Errorf("expected prefix todo_auth_, got %q", name)
		}
		if !strings.HasSuffix(name, ".md") {
			t.Errorf("expected .md suffix, got %q", name)
		}
	})

	t.Run("no tags uses only timestamp", func(t *testing.T) {
		name := buildFilename(nil, "2026-02-27T14-30-00")
		if name != "2026-02-27T14-30-00.md" {
			t.Errorf("unexpected filename: %q", name)
		}
	})
}

func TestWriteNotesFile(t *testing.T) {
	notes := []store.Note{
		{ID: 1, Text: "fix the login bug", Tags: "#auth #bug", CreatedAt: "2026-02-27 10:00:00"},
		{ID: 2, Text: "refactor middleware", Tags: "#auth", CreatedAt: "2026-02-27 11:00:00"},
	}

	dir := t.TempDir()
	path, err := writeNotesFile(notes, []string{"#auth"}, dir, "2026-02-27T14-30-00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, "auth_2026-02-27T14-30-00.md")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read output file: %v", err)
	}

	body := string(content)

	if !strings.Contains(body, "fix the login bug") {
		t.Errorf("expected first note text in output")
	}
	if !strings.Contains(body, "refactor middleware") {
		t.Errorf("expected second note text in output")
	}
	if !strings.Contains(body, "---") {
		t.Errorf("expected --- delimiter between notes")
	}
	if !strings.Contains(body, "_2026-02-27 10:00:00") {
		t.Errorf("expected created_at metadata in output")
	}
}

func TestWriteNotesFileCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "exports")
	notes := []store.Note{
		{ID: 1, Text: "a note", Tags: "#todo", CreatedAt: "2026-02-27 10:00:00"},
	}
	_, err := writeNotesFile(notes, []string{"#todo"}, dir, "2026-02-27T14-30-00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected directory to be created at %s", dir)
	}
}
