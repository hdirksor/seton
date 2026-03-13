package config

// Platform abstracts OS-specific behavior for config resolution.
type Platform interface {
	// DefaultEditor returns the fallback editor when none is configured.
	DefaultEditor() string
	// AppDir returns the directory where seton stores config and data files.
	AppDir() (string, error)
	// ExpandTilde expands a leading ~ to the user's home directory.
	ExpandTilde(path string) string
}

// platform is the active implementation, set by platform_unix.go or platform_windows.go.
var platform Platform
