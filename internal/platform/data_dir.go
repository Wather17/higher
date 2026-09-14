package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

var ErrHomeDirectoryUnavailable = errors.New("home directory is unavailable")

// DefaultDataDir returns the directory where Higher stores user data.
func DefaultDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	return ResolveDataDir(runtime.GOOS, os.Getenv, homeDir)
}

// ResolveDataDir resolves a platform data directory using the supplied environment.
// It is kept separate from DefaultDataDir so path behavior can be tested without
// mutating the host process environment.
func ResolveDataDir(goos string, getenv func(string) string, homeDir string) (string, error) {
	var baseDir string

	switch goos {
	case "linux":
		baseDir = getenv("XDG_DATA_HOME")
		if baseDir == "" {
			if homeDir == "" {
				return "", ErrHomeDirectoryUnavailable
			}
			baseDir = filepath.Join(homeDir, ".local", "share")
		}
	case "darwin":
		if homeDir == "" {
			return "", ErrHomeDirectoryUnavailable
		}
		baseDir = filepath.Join(homeDir, "Library", "Application Support")
	case "windows":
		baseDir = getenv("LOCALAPPDATA")
		if baseDir == "" {
			if homeDir == "" {
				return "", ErrHomeDirectoryUnavailable
			}
			baseDir = filepath.Join(homeDir, "AppData", "Local")
		}
	default:
		if homeDir == "" {
			return "", ErrHomeDirectoryUnavailable
		}
		baseDir = filepath.Join(homeDir, ".higher")
	}

	return filepath.Join(baseDir, "higher"), nil
}

// EnsureDataDir creates and returns the default data directory with private
// permissions suitable for personal financial data.
func EnsureDataDir() (string, error) {
	directory, err := DefaultDataDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}

	return directory, nil
}
