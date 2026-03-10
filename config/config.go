package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

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
	if strings.HasPrefix(p.Root, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p.Root
		}
		return filepath.Join(home, p.Root[2:])
	}
	return p.Root
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

// Load reads ~/.seton/config.toml and returns the result merged with defaults.
// If the file does not exist, defaults are returned with no error.
func Load() (Config, error) {
	cfg := defaults()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, err
	}

	path := filepath.Join(home, ".seton", "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	_, err = toml.DecodeFile(path, &cfg)
	return cfg, err
}
