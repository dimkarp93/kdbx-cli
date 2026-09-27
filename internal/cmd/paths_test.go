package cmd

import (
	"path/filepath"
	"testing"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func TestPathEntriesListsKeyStores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	path := config.DefaultPath()
	cfg := config.Config{Sections: map[string]config.Section{
		"work":    {KeyStore: "~/work.kdbx"},
		"default": {KeyStore: filepath.Join(home, "default.kdbx")},
		"empty":   {},
	}}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	entries := pathEntries()
	if len(entries) != 3 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	if entries[0].Name != "config" || entries[0].Path != path {
		t.Fatalf("config entry: %+v", entries[0])
	}
	if entries[1].Name != "key-store:default" || entries[1].Path != filepath.Join(home, "default.kdbx") {
		t.Fatalf("default entry: %+v", entries[1])
	}
	if entries[2].Name != "key-store:work" || entries[2].Path != filepath.Join(home, "work.kdbx") {
		t.Fatalf("work entry: %+v", entries[2])
	}
}

func TestPathEntriesWithoutAConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	entries := pathEntries()
	if len(entries) != 1 || entries[0].Name != "config" {
		t.Fatalf("got %+v", entries)
	}
}
