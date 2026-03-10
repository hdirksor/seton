package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- parseBlocks ---

func TestParseBlocks(t *testing.T) {
	t.Run("empty content returns empty slice", func(t *testing.T) {
		got := parseBlocks("", "~!", "!~")
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("no delimiters returns empty slice", func(t *testing.T) {
		got := parseBlocks("hello world", "~!", "!~")
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("single block", func(t *testing.T) {
		got := parseBlocks("~! hello world !~", "~!", "!~")
		if len(got) != 1 {
			t.Fatalf("expected 1, got %d", len(got))
		}
		if got[0] != "hello world" {
			t.Errorf("expected %q, got %q", "hello world", got[0])
		}
	})

	t.Run("multiple blocks", func(t *testing.T) {
		content := "~! first note !~ some code ~! second note !~"
		got := parseBlocks(content, "~!", "!~")
		if len(got) != 2 {
			t.Fatalf("expected 2, got %d", len(got))
		}
		if got[0] != "first note" {
			t.Errorf("expected %q, got %q", "first note", got[0])
		}
		if got[1] != "second note" {
			t.Errorf("expected %q, got %q", "second note", got[1])
		}
	})

	t.Run("whitespace trimmed from blocks", func(t *testing.T) {
		got := parseBlocks("~!   spaces   !~", "~!", "!~")
		if len(got) != 1 || got[0] != "spaces" {
			t.Errorf("expected %q, got %v", "spaces", got)
		}
	})

	t.Run("multiline block", func(t *testing.T) {
		got := parseBlocks("~!\nline one\nline two\n!~", "~!", "!~")
		if len(got) != 1 {
			t.Fatalf("expected 1, got %d", len(got))
		}
		if !strings.Contains(got[0], "line one") || !strings.Contains(got[0], "line two") {
			t.Errorf("expected multiline content, got %q", got[0])
		}
	})

	t.Run("unclosed delimiter ignored", func(t *testing.T) {
		got := parseBlocks("~! no close here", "~!", "!~")
		if len(got) != 0 {
			t.Errorf("expected empty for unclosed block, got %v", got)
		}
	})

	t.Run("empty block ignored", func(t *testing.T) {
		got := parseBlocks("~!   !~", "~!", "!~")
		if len(got) != 0 {
			t.Errorf("expected empty block to be ignored, got %v", got)
		}
	})
}

// --- archiveFile ---

func TestArchiveFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	archiveDir := t.TempDir()
	date := "2026-01-02"

	if err := archiveFile(src, archiveDir, date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// original file should be gone
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("expected original file to be removed")
	}

	// archived file should exist with date prefix
	expected := filepath.Join(archiveDir, "2026-01-02_notes.md")
	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("expected archived file at %s: %v", expected, err)
	}

	// file should be read-only
	if info.Mode()&0222 != 0 {
		t.Errorf("expected file to be read-only, got mode %v", info.Mode())
	}
}

func TestArchiveFileCreatesDir(t *testing.T) {
	src := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	archiveDir := filepath.Join(t.TempDir(), "nested", "archive")

	if err := archiveFile(src, archiveDir, "2026-01-02"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(archiveDir); err != nil {
		t.Errorf("expected archive dir to be created: %v", err)
	}
}

// --- importModel ---

func TestImportModelInitialState(t *testing.T) {
	blocks := []string{"first block #todo", "second block #auth"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })

	if m.current != 0 {
		t.Errorf("expected current=0, got %d", m.current)
	}
	if m.completed {
		t.Error("expected completed=false")
	}
	// tag input should be pre-populated with extracted tags
	if !strings.Contains(m.tagInput.Value(), "#todo") {
		t.Errorf("expected tag input to contain #todo, got %q", m.tagInput.Value())
	}
}

func TestImportModelCtrlS(t *testing.T) {
	var savedText string
	blocks := []string{"note #todo", "second"}
	m := newImportModel(blocks, func(text, _ string) error {
		savedText = text
		return nil
	})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	updated := result.(importModel)

	if savedText != "note #todo" {
		t.Errorf("expected saved text %q, got %q", "note #todo", savedText)
	}
	if updated.saved != 1 {
		t.Errorf("expected saved=1, got %d", updated.saved)
	}
	if updated.current != 1 {
		t.Errorf("expected current=1, got %d", updated.current)
	}
}

func TestImportModelCtrlK(t *testing.T) {
	blocks := []string{"first", "second"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updated := result.(importModel)

	if updated.skipped != 1 {
		t.Errorf("expected skipped=1, got %d", updated.skipped)
	}
	if updated.current != 1 {
		t.Errorf("expected current=1 after skip, got %d", updated.current)
	}
}

func TestImportModelCompletesAfterLastBlock(t *testing.T) {
	blocks := []string{"only block"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	updated := result.(importModel)

	if !updated.completed {
		t.Error("expected completed=true after last block saved")
	}
}

func TestImportModelSkipCompletesAfterLastBlock(t *testing.T) {
	blocks := []string{"only block"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updated := result.(importModel)

	if !updated.completed {
		t.Error("expected completed=true after last block skipped")
	}
}

func TestImportModelCtrlC(t *testing.T) {
	blocks := []string{"first", "second"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated := result.(importModel)

	if !updated.quitting {
		t.Error("expected quitting=true after ctrl+c")
	}
	if updated.completed {
		t.Error("expected completed=false when quitting early")
	}
}

func TestImportModelSaveError(t *testing.T) {
	blocks := []string{"first"}
	m := newImportModel(blocks, func(_, _ string) error {
		return errors.New("db error")
	})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	updated := result.(importModel)

	if updated.saveErr == nil {
		t.Error("expected saveErr to be set after save failure")
	}
	if updated.saved != 0 {
		t.Errorf("expected saved=0 after save failure, got %d", updated.saved)
	}
	// current should NOT advance on save error
	if updated.current != 0 {
		t.Errorf("expected current=0 after save failure, got %d", updated.current)
	}
}

func TestArchiveFileMkdirError(t *testing.T) {
	src := filepath.Join(t.TempDir(), "notes.md")
	os.WriteFile(src, []byte("content"), 0644)
	// Place a regular file where the archive dir would need to be created.
	blocker := filepath.Join(t.TempDir(), "notadir")
	os.WriteFile(blocker, []byte("x"), 0644)
	err := archiveFile(src, filepath.Join(blocker, "subdir"), "2026-01-02")
	if err == nil {
		t.Error("expected error when archive dir cannot be created")
	}
}

func TestArchiveFileRenameError(t *testing.T) {
	archiveDir := t.TempDir()
	err := archiveFile("/nonexistent/path/file.md", archiveDir, "2026-01-02")
	if err == nil {
		t.Error("expected error when source file does not exist")
	}
}

func TestImportModelInit(t *testing.T) {
	m := newImportModel([]string{"block"}, func(_, _ string) error { return nil })
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected Init to return a non-nil cmd")
	}
}

func TestImportModelNonKeyMessage(t *testing.T) {
	blocks := []string{"first #todo"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })
	// Sending a non-key message should be forwarded to tagInput, not crash
	result, cmd := m.Update("not a key message")
	updated := result.(importModel)
	if updated.current != 0 {
		t.Errorf("expected current unchanged, got %d", updated.current)
	}
	_ = cmd // cmd comes from tagInput.Update, just verify no panic
}

func TestImportModelView(t *testing.T) {
	t.Run("quitting shows abort message", func(t *testing.T) {
		m := newImportModel([]string{"block"}, func(_, _ string) error { return nil })
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		updated := result.(importModel)
		view := updated.View()
		if !strings.Contains(view, "Aborted") {
			t.Errorf("expected 'Aborted' in view, got: %s", view)
		}
	})

	t.Run("completed shows done message", func(t *testing.T) {
		m := newImportModel([]string{"block"}, func(_, _ string) error { return nil })
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		updated := result.(importModel)
		view := updated.View()
		if !strings.Contains(view, "Done") {
			t.Errorf("expected 'Done' in view, got: %s", view)
		}
	})

	t.Run("mid-session shows block and hints", func(t *testing.T) {
		m := newImportModel([]string{"first block", "second block"}, func(_, _ string) error { return nil })
		view := m.View()
		if !strings.Contains(view, "Block 1 / 2") {
			t.Errorf("expected block counter in view, got: %s", view)
		}
		if !strings.Contains(view, "first block") {
			t.Errorf("expected block text in view, got: %s", view)
		}
		if !strings.Contains(view, "ctrl+s save") {
			t.Errorf("expected hints in view, got: %s", view)
		}
	})

	t.Run("mid-session with save error shows error", func(t *testing.T) {
		m := newImportModel([]string{"block"}, func(_, _ string) error { return errors.New("db error") })
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		updated := result.(importModel)
		view := updated.View()
		if !strings.Contains(view, "Error:") {
			t.Errorf("expected error in view, got: %s", view)
		}
	})

	t.Run("past end of blocks returns empty", func(t *testing.T) {
		m := newImportModel([]string{"block"}, func(_, _ string) error { return nil })
		m.current = 99 // manually push past end without completing
		view := m.View()
		if view != "" {
			t.Errorf("expected empty view when past end, got: %s", view)
		}
	})
}

func TestImportModelTagInputUpdatesOnAdvance(t *testing.T) {
	blocks := []string{"first #todo", "second #auth"}
	m := newImportModel(blocks, func(_, _ string) error { return nil })

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updated := result.(importModel)

	if !strings.Contains(updated.tagInput.Value(), "#auth") {
		t.Errorf("expected tag input to contain #auth after advancing, got %q", updated.tagInput.Value())
	}
}

func fixedProg(m importModel) func(tea.Model) (tea.Model, error) {
	return func(_ tea.Model) (tea.Model, error) { return m, nil }
}

func TestRunImport(t *testing.T) {
	t.Run("file not found returns error", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		err := runImport(nil, []string{"/nonexistent/file.md"})
		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("no blocks returns error", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		f := filepath.Join(t.TempDir(), "notes.md")
		os.WriteFile(f, []byte("no delimiters here"), 0644)
		err := runImport(nil, []string{f})
		if err == nil {
			t.Error("expected error when no blocks found")
		}
	})

	t.Run("runProg error is returned", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		f := filepath.Join(t.TempDir(), "notes.md")
		os.WriteFile(f, []byte("~! a note !~"), 0644)
		stub := func(_ tea.Model) (tea.Model, error) { return nil, errors.New("terminal error") }
		if err := runImport(stub, []string{f}); err == nil {
			t.Error("expected error from runProg")
		}
	})

	t.Run("completed archives the file", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		f := filepath.Join(t.TempDir(), "notes.md")
		os.WriteFile(f, []byte("~! a note !~"), 0644)
		if err := runImport(fixedProg(importModel{completed: true, saved: 1}), []string{f}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Error("expected original file to be gone after archive")
		}
	})

	t.Run("quitting does not archive the file", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		f := filepath.Join(t.TempDir(), "notes.md")
		os.WriteFile(f, []byte("~! a note !~"), 0644)
		if err := runImport(fixedProg(importModel{quitting: true}), []string{f}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected original file to still exist: %v", err)
		}
	})

	t.Run("store open error is returned", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// Place a regular file at ~/.seton so MkdirAll inside store.Open fails.
		os.WriteFile(filepath.Join(home, ".seton"), []byte("x"), 0644)
		f := filepath.Join(t.TempDir(), "notes.md")
		os.WriteFile(f, []byte("~! a note !~"), 0644)
		if err := runImport(nil, []string{f}); err == nil {
			t.Error("expected error from store.Open")
		}
	})

	t.Run("archiveFile error is returned", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// Place a regular file at the archive path so MkdirAll inside archiveFile fails.
		os.MkdirAll(filepath.Join(home, "seton", "notes"), 0755)
		os.WriteFile(filepath.Join(home, "seton", "notes", ".archived"), []byte("x"), 0644)
		f := filepath.Join(t.TempDir(), "notes.md")
		os.WriteFile(f, []byte("~! a note !~"), 0644)
		if err := runImport(fixedProg(importModel{completed: true}), []string{f}); err == nil {
			t.Error("expected error from archiveFile")
		}
	})
}
