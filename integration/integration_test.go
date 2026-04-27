package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdirksor/seton/commands"
	"github.com/hdirksor/seton/store"
)

// rootCmd builds a fresh cobra root and redirects output to the returned buffer.
func rootCmd(t *testing.T, args ...string) (stdout *bytes.Buffer, err error) {
	t.Helper()
	var buf bytes.Buffer
	cmd := commands.InitRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return &buf, err
}

// seedNote saves a note directly via the store, bypassing the TUI.
func seedNote(t *testing.T, text, tags string) {
	t.Helper()
	db, err := store.Open()
	if err != nil {
		t.Fatalf("seedNote: open db: %v", err)
	}
	defer db.Close()
	if err := store.SaveNote(db, text, tags); err != nil {
		t.Fatalf("seedNote: save: %v", err)
	}
}

func TestQueryEmptyDB(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	out, err := rootCmd(t, "query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "No notes found") {
		t.Errorf("expected 'No notes found', got: %q", out.String())
	}
}

func TestQueryAllNotes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedNote(t, "first note", "#alpha")
	seedNote(t, "second note", "#beta")

	out, err := rootCmd(t, "query")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "first note") {
		t.Errorf("expected 'first note' in output, got: %q", s)
	}
	if !strings.Contains(s, "second note") {
		t.Errorf("expected 'second note' in output, got: %q", s)
	}
}

func TestQueryByTag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedNote(t, "tagged note", "#mytag")
	seedNote(t, "other note", "#other")

	out, err := rootCmd(t, "query", "#mytag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "tagged note") {
		t.Errorf("expected 'tagged note' in output, got: %q", s)
	}
	if strings.Contains(s, "other note") {
		t.Errorf("expected 'other note' to be excluded, got: %q", s)
	}
}

func TestQueryByTagNoMatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedNote(t, "a note", "#alpha")

	out, err := rootCmd(t, "query", "#notexist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "No notes found") {
		t.Errorf("expected 'No notes found', got: %q", out.String())
	}
}

func TestQueryNormalizeTag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedNote(t, "note without hash", "#todo")

	// pass tag without leading # — normalizeTags should add it
	out, err := rootCmd(t, "query", "todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "note without hash") {
		t.Errorf("expected note in output when tag given without #, got: %q", out.String())
	}
}

func TestExportWritesFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedNote(t, "exportable note", "#export")

	dir := t.TempDir()
	_, err := rootCmd(t, "export", "--dir", dir, "#export")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading export dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in export dir, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("reading exported file: %v", err)
	}
	if !strings.Contains(string(content), "exportable note") {
		t.Errorf("expected note text in exported file, got: %q", string(content))
	}
}

func TestExportNoNotes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := rootCmd(t, "export", "#notexist")
	if err == nil {
		t.Error("expected error when no notes match, got nil")
	}
}
