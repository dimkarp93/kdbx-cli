package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRaw(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cfg")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMigrateV0ToV1InvertsSecrets(t *testing.T) {
	path := writeRaw(t, `{"sections":{"default":{"key-store":"/k.kdbx","secrets":{"Token":"GH_TOKEN","Pw":"DB_PASSWORD"}},"psql":{"secrets":{"Pw":"PGPASSWORD"}}},"cached":{"enabled":true,"ttl":"5m"}}`)
	if err := Migrate(path, 0, 1); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	def := c.Sections["default"]
	if def.KeyStore != "/k.kdbx" || def.Secrets["GH_TOKEN"] != "Token" || def.Secrets["DB_PASSWORD"] != "Pw" || len(def.Secrets) != 2 {
		t.Errorf("default: %+v", def)
	}
	if c.Sections["psql"].Secrets["PGPASSWORD"] != "Pw" {
		t.Errorf("psql: %+v", c.Sections["psql"])
	}
	if c.Cache == nil || !c.Cache.Enabled || c.Cache.TTL != "5m" {
		t.Errorf("cache: %+v", c.Cache)
	}
}

func TestMigrateCollisionLeavesFileUntouched(t *testing.T) {
	body := `{"sections":{"default":{"secrets":{"A":"X","B":"X"}}}}`
	path := writeRaw(t, body)
	err := Migrate(path, 0, 1)
	if err == nil || !strings.Contains(err.Error(), `"X"`) {
		t.Fatalf("err=%v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != body {
		t.Errorf("file changed: %s", data)
	}
}

func TestMigrateAlreadyAtTarget(t *testing.T) {
	path := writeRaw(t, `{"version":1,"sections":{}}`)
	if err := Migrate(path, 0, 1); !errors.Is(err, ErrAlreadyMigrated) {
		t.Errorf("err=%v", err)
	}
}

func TestMigrateFromMismatch(t *testing.T) {
	path := writeRaw(t, `{"version":1,"sections":{}}`)
	if err := Migrate(path, 0, 2); err == nil {
		t.Error("expected error for unsupported range")
	}
	path = writeRaw(t, `{"sections":{}}`)
	err := Migrate(path, 1, 1)
	if err == nil || errors.Is(err, ErrAlreadyMigrated) {
		t.Errorf("err=%v", err)
	}
}
