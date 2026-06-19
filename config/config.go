// Package config handles loading and resolving seton configuration.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// AppDir returns the directory where seton stores config and data files.
func AppDir() (string, error) {
	return platform.AppDir()
}

// ResolveEditor returns the editor to use when opening files. It checks the
// config value first, then $EDITOR, then falls back to the platform default.
func (c Config) ResolveEditor() string {
	if c.Editor != "" {
		return c.Editor
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return platform.DefaultEditor()
}

// Delimiters holds the open/close markers used to identify notes in a file.
type Delimiters struct {
	Open  string `toml:"open"`
	Close string `toml:"close"`
}

// Paths holds directory locations used by seton.
// Root defaults to ~/seton. All other paths are derived from it.
type Paths struct {
	Root string `toml:"root"`
}

// expandRoot resolves ~ in Root to the user's home directory.
func (p Paths) expandRoot() string {
	return platform.ExpandTilde(p.Root)
}

// Notes returns the directory where the user writes source note files.
func (p Paths) Notes() string {
	return filepath.Join(p.expandRoot(), "notes")
}

// Exports returns the directory where seton writes exported markdown files.
func (p Paths) Exports() string {
	return filepath.Join(p.expandRoot(), "exports")
}

// Archive returns the directory where imported files are moved after processing.
func (p Paths) Archive() string {
	return filepath.Join(p.expandRoot(), "notes", ".archived")
}

// Config holds all seton configuration.
type Config struct {
	Delimiters Delimiters `toml:"delimiters"`
	Paths      Paths      `toml:"paths"`
	Editor     string     `toml:"editor"`
}

func defaults() Config {
	return Config{
		Delimiters: Delimiters{
			Open:  "~!",
			Close: "!~",
		},
		Paths: Paths{
			Root: "~/seton",
		},
	}
}

// Load reads config.toml from the platform app dir and returns the result merged with defaults.
// If the file does not exist, defaults are returned with no error.
func Load() (Config, error) {
	cfg := defaults()

	dir, err := platform.AppDir()
	if err != nil {
		return cfg, err
	}

	path := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	_, err = toml.DecodeFile(path, &cfg)
	return cfg, err
}
