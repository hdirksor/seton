//go:build windows

package config

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	platform = windowsPlatform{}
}

type windowsPlatform struct{}

func (windowsPlatform) DefaultEditor() string { return "notepad" }

func (windowsPlatform) AppDir() (string, error) {
	appdata, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appdata, "seton"), nil
}

func (windowsPlatform) ExpandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
