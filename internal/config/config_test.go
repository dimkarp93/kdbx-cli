package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg")
	if err := os.WriteFile(path, []byte(`{"version":1,"sections":{"default":{"key-store":"/k.kdbx"}},"cached":{"enabled":true,"ttl":"5m"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sections["default"].KeyStore != "/k.kdbx" {
		t.Errorf("sections: %+v", c.Sections)
	}
	if c.Cache == nil || !c.Cache.Enabled || c.Cache.TTL != "5m" {
		t.Errorf("cache: %+v", c.Cache)
	}
}

func TestSaveReloadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg")
	c := Config{Sections: map[string]Section{"default": {KeyStore: "/k.kdbx"}}}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"sections"`) {
		t.Errorf("saved file should contain sections:\n%s", data)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Sections["default"].KeyStore != "/k.kdbx" {
		t.Errorf("reload after save: %+v", c2.Sections)
	}
}

func TestSaveReloadRoundTripWithTemplates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg")
	c := Config{Sections: map[string]Section{
		"default": {
			KeyStore:  "/k.kdbx",
			Templates: []Template{{Name: "cfg", Path: "/tmp/settings.json.tmpl"}},
		},
	}}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	c2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got := c2.Sections["default"].Templates
	if len(got) != 1 || got[0].Name != "cfg" || got[0].Path != "/tmp/settings.json.tmpl" {
		t.Errorf("templates after roundtrip: %+v", got)
	}
}

func TestLoadRejectsUnexpectedVersion(t *testing.T) {
	for _, body := range []string{`{"sections":{}}`, `{"version":0,"sections":{}}`, `{"version":2,"sections":{}}`} {
		path := filepath.Join(t.TempDir(), "cfg")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if err == nil || !strings.Contains(err.Error(), "kdbx-cli migrate") {
			t.Errorf("%s: err=%v", body, err)
		}
	}
}

func TestSaveWritesCurrentVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg")
	if err := Save(path, Config{}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"version": 1`) {
		t.Errorf("version missing:\n%s", data)
	}
}

func TestPermissionWarnings(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "default")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if w := PermissionWarnings(path); len(w) != 0 {
		t.Errorf("private config: %v", w)
	}
	if err := os.Chmod(path, 0666); err != nil {
		t.Fatal(err)
	}
	if w := PermissionWarnings(path); len(w) != 1 {
		t.Errorf("world-writable file: %v", w)
	}
	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatal(err)
	}
	if w := PermissionWarnings(path); len(w) != 2 {
		t.Errorf("world-writable file and dir: %v", w)
	}
	if w := PermissionWarnings(filepath.Join(dir, "missing")); len(w) != 1 {
		t.Errorf("missing file in a world-writable dir: %v", w)
	}
}
