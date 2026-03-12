package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"
)

func TestJotCmd(t *testing.T) {
	t.Run("has correct use", func(t *testing.T) {
		cmd := jotCmd()
		if cmd.Use != "jot" {
			t.Errorf("expected Use %q, got %q", "jot", cmd.Use)
		}
	})

	t.Run("rejects extra args", func(t *testing.T) {
		cmd := jotCmd()
		cmd.SetArgs([]string{"unexpected"})
		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when args provided, got nil")
		}
	})
}

func TestJotModelCtrlE(t *testing.T) {
	m := newJotModel()
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	updated := result.(jotModel)

	if !updated.toEditor {
		t.Error("expected toEditor=true after ctrl+e")
	}
	if cmd == nil {
		t.Error("expected quit cmd from ctrl+e")
	}
}

func TestJotModelCtrlEPreservesNoteText(t *testing.T) {
	m := newJotModel()
	*m.noteText = "some draft text"

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	updated := result.(jotModel)

	if *updated.noteText != "some draft text" {
		t.Errorf("expected noteText preserved, got %q", *updated.noteText)
	}
}

func TestJotModelNonCtrlEForwardedToForm(t *testing.T) {
	m := newJotModel()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := result.(jotModel)

	if updated.toEditor {
		t.Error("expected toEditor=false for non ctrl+e key")
	}
}

func TestJotNoteValidation(t *testing.T) {
	t.Run("empty note is rejected", func(t *testing.T) {
		m := newJotModel()
		// form starts on the note group; attempt to advance with empty text
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		updated := result.(jotModel)
		if updated.form.State == huh.StateCompleted {
			t.Error("expected form to remain incomplete when note is empty")
		}
	})

	t.Run("whitespace-only note is rejected", func(t *testing.T) {
		m := newJotModel()
		*m.noteText = "   "
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
		updated := result.(jotModel)
		if updated.form.State == huh.StateCompleted {
			t.Error("expected form to remain incomplete when note is whitespace only")
		}
	})
}

func TestWriteNoteFile(t *testing.T) {
	dir := t.TempDir()
	path, err := writeNoteFile("my note text", "~!", "!~", dir, "2026-03-10T12-00-00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != filepath.Join(dir, "2026-03-10T12-00-00.md") {
		t.Errorf("unexpected path: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read written file: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "~!") || !strings.Contains(s, "!~") {
		t.Errorf("expected delimiters in file content, got: %s", s)
	}
	if !strings.Contains(s, "my note text") {
		t.Errorf("expected note text in file content, got: %s", s)
	}
}

func TestWriteNoteFileTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	_, err := writeNoteFile("  padded  ", "~!", "!~", dir, "2026-03-10T12-00-00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "2026-03-10T12-00-00.md"))
	if strings.Contains(string(content), "  padded  ") {
		t.Error("expected leading/trailing whitespace to be trimmed")
	}
}

func TestWriteNoteFileCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "notes")
	_, err := writeNoteFile("text", "~!", "!~", dir, "2026-03-10T12-00-00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected notes dir to be created: %v", err)
	}
}

func TestWriteNoteFileMkdirError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "notadir")
	os.WriteFile(blocker, []byte("x"), 0644)
	_, err := writeNoteFile("text", "~!", "!~", filepath.Join(blocker, "notes"), "2026-03-10T12-00-00")
	if err == nil {
		t.Error("expected error when notes dir cannot be created")
	}
}
