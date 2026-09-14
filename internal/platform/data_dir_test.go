package platform

import (
	"path/filepath"
	"testing"
)

func TestResolveDataDirLinuxUsesXDGDataHome(t *testing.T) {
	getenv := func(key string) string {
		if key == "XDG_DATA_HOME" {
			return "/tmp/user-data"
		}
		return ""
	}

	directory, err := ResolveDataDir("linux", getenv, "/home/tester")
	if err != nil {
		t.Fatalf("resolving data directory: %v", err)
	}

	expected := filepath.Join("/tmp/user-data", "higher")
	if directory != expected {
		t.Fatalf("expected %q, got %q", expected, directory)
	}
}

func TestResolveDataDirLinuxFallsBackToHome(t *testing.T) {
	directory, err := ResolveDataDir("linux", func(string) string { return "" }, "/home/tester")
	if err != nil {
		t.Fatalf("resolving data directory: %v", err)
	}

	expected := filepath.Join("/home/tester", ".local", "share", "higher")
	if directory != expected {
		t.Fatalf("expected %q, got %q", expected, directory)
	}
}

func TestResolveDataDirMacOSUsesApplicationSupport(t *testing.T) {
	directory, err := ResolveDataDir("darwin", func(string) string { return "" }, "/Users/tester")
	if err != nil {
		t.Fatalf("resolving data directory: %v", err)
	}

	expected := filepath.Join("/Users/tester", "Library", "Application Support", "higher")
	if directory != expected {
		t.Fatalf("expected %q, got %q", expected, directory)
	}
}

func TestResolveDataDirWindowsUsesLocalAppData(t *testing.T) {
	getenv := func(key string) string {
		if key == "LOCALAPPDATA" {
			return `C:\Users\tester\AppData\Local`
		}
		return ""
	}

	directory, err := ResolveDataDir("windows", getenv, `C:\Users\tester`)
	if err != nil {
		t.Fatalf("resolving data directory: %v", err)
	}

	expected := filepath.Join(`C:\Users\tester\AppData\Local`, "higher")
	if directory != expected {
		t.Fatalf("expected %q, got %q", expected, directory)
	}
}

func TestResolveDataDirRequiresHomeWhenNoPlatformOverrideExists(t *testing.T) {
	if _, err := ResolveDataDir("linux", func(string) string { return "" }, ""); err != ErrHomeDirectoryUnavailable {
		t.Fatalf("expected ErrHomeDirectoryUnavailable, got %v", err)
	}
}
