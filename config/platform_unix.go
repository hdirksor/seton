//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	platform = unixPlatform{}
}

type unixPlatform struct{}

func (unixPlatform) DefaultEditor() string { return "vi" }

func (unixPlatform) AppDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".seton"), nil
}

func (unixPlatform) ExpandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
