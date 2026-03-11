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

func TestResolveEditor(t *testing.T) {
	t.Run("uses config editor when set", func(t *testing.T) {
		cfg := Config{Editor: "nano"}
		if got := cfg.ResolveEditor(); got != "nano" {
			t.Errorf("expected nano, got %s", got)
		}
	})

	t.Run("falls back to EDITOR env var", func(t *testing.T) {
		t.Setenv("EDITOR", "emacs")
		cfg := Config{}
		if got := cfg.ResolveEditor(); got != "emacs" {
			t.Errorf("expected emacs, got %s", got)
		}
	})

	t.Run("falls back to vi when neither set", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		cfg := Config{}
		if got := cfg.ResolveEditor(); got != "vi" {
			t.Errorf("expected vi, got %s", got)
		}
	})

	t.Run("config editor takes precedence over EDITOR env var", func(t *testing.T) {
		t.Setenv("EDITOR", "emacs")
		cfg := Config{Editor: "nano"}
		if got := cfg.ResolveEditor(); got != "nano" {
			t.Errorf("expected nano, got %s", got)
		}
	})
}

func TestEditorConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".seton"), 0755)
	os.WriteFile(filepath.Join(dir, ".seton", "config.toml"), []byte(`
editor = "code --wait"
`), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Editor != "code --wait" {
		t.Errorf("expected editor %q, got %q", "code --wait", cfg.Editor)
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
