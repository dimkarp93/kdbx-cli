package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	t.Setenv("HOME", "/h")
	if got := ExpandHome("~/x"); got != "/h/x" {
		t.Fatalf("got %q", got)
	}
}

func TestDirFollowsXDG(t *testing.T) {
	t.Setenv("HOME", "/h")
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := Dir(); got != "/h/.config/kdbx-cli" {
		t.Fatalf("got %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if got := Dir(); got != "/x/kdbx-cli" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultPathFallsBackToTheLegacyFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))

	legacy := filepath.Join(home, ".config", "kdbx-cli", "default")
	newPath := filepath.Join(home, "xdg", "kdbx-cli", "default")

	if got := DefaultPath(); got != newPath {
		t.Fatalf("nothing exists: got %q, want %q", got, newPath)
	}

	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := DefaultPath(); got != legacy {
		t.Fatalf("legacy exists: got %q, want %q", got, legacy)
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := DefaultPath(); got != newPath {
		t.Fatalf("both exist: got %q, want %q", got, newPath)
	}
}
