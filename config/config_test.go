package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Delimiters.Open != "~!" {
		t.Errorf("expected default open %q, got %q", "~!", cfg.Delimiters.Open)
	}
	if cfg.Delimiters.Close != "!~" {
		t.Errorf("expected default close %q, got %q", "!~", cfg.Delimiters.Close)
	}
	if cfg.Paths.Notes() != filepath.Join(home, "seton", "notes") {
		t.Errorf("unexpected default notes path: %q", cfg.Paths.Notes())
	}
	if cfg.Paths.Exports() != filepath.Join(home, "seton", "exports") {
		t.Errorf("unexpected default exports path: %q", cfg.Paths.Exports())
	}
	if cfg.Paths.Archive() != filepath.Join(home, "seton", "notes", ".archived") {
		t.Errorf("unexpected default archive path: %q", cfg.Paths.Archive())
	}
}

func TestLoadReadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".seton"), 0755)
	os.WriteFile(filepath.Join(dir, ".seton", "config.toml"), []byte(`
[delimiters]
open  = "<<"
close = ">>"
`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Delimiters.Open != "<<" {
		t.Errorf("expected open %q, got %q", "<<", cfg.Delimiters.Open)
	}
	if cfg.Delimiters.Close != ">>" {
		t.Errorf("expected close %q, got %q", ">>", cfg.Delimiters.Close)
	}
}

func TestLoadPartialConfigKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".seton"), 0755)
	// Only override open — close should remain the default.
	os.WriteFile(filepath.Join(dir, ".seton", "config.toml"), []byte(`
[delimiters]
open = "<<"
`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Delimiters.Open != "<<" {
		t.Errorf("expected open %q, got %q", "<<", cfg.Delimiters.Open)
	}
	if cfg.Delimiters.Close != "!~" {
		t.Errorf("expected default close %q, got %q", "!~", cfg.Delimiters.Close)
	}
}

func TestCustomRootPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".seton"), 0755)
	os.WriteFile(filepath.Join(dir, ".seton", "config.toml"), []byte(`
[paths]
root = "/tmp/my-notes"
`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Paths.Notes() != "/tmp/my-notes/notes" {
		t.Errorf("unexpected notes path: %q", cfg.Paths.Notes())
	}
	if cfg.Paths.Exports() != "/tmp/my-notes/exports" {
		t.Errorf("unexpected exports path: %q", cfg.Paths.Exports())
	}
	if cfg.Paths.Archive() != "/tmp/my-notes/notes/.archived" {
		t.Errorf("unexpected archive path: %q", cfg.Paths.Archive())
	}
}

func TestTildeExpansion(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".seton"), 0755)
	os.WriteFile(filepath.Join(dir, ".seton", "config.toml"), []byte(`
[paths]
root = "~/custom"
`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Paths.Notes() != filepath.Join(dir, "custom", "notes") {
		t.Errorf("unexpected notes path: %q", cfg.Paths.Notes())
	}
}
