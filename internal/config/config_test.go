package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRaw(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cfg")
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}

func TestLoadConfig(t *testing.T) {
	path := writeRaw(t, `{"version":1,"sections":{"default":{"key-store":"/k.kdbx"}},"cached":{"enabled":true,"ttl":"5m"}}`)

	c, err := Load(path)

	t.Run("succeeds", func(t *testing.T) {
		require.NoError(t, err)
	})
	t.Run("sections", func(t *testing.T) {
		assert.Equal(t, map[string]Section{"default": {KeyStore: "/k.kdbx"}}, c.Sections)
	})
	t.Run("cache", func(t *testing.T) {
		assert.Equal(t, &CacheConfig{Enabled: true, TTL: "5m"}, c.Cache)
	})
}

func TestLoadRejectsUnexpectedVersion(t *testing.T) {
	cases := map[string]string{
		"no version": `{"sections":{}}`,
		"version 0":  `{"version":0,"sections":{}}`,
		"version 2":  `{"version":2,"sections":{}}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Load(writeRaw(t, body))
			assert.ErrorContains(t, err, "kdbx-cli migrate")
		})
	}
}

func TestSaveWritesCurrentVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg")
	require.NoError(t, Save(path, Config{}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), fmt.Sprintf(`"version": %d`, CurrentVersion))
}

func permissionFixture(t *testing.T, dirMode, fileMode os.FileMode) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "default")
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0600))
	require.NoError(t, os.Chmod(path, fileMode))
	require.NoError(t, os.Chmod(dir, dirMode))

	return path
}

func TestPermissionWarningsPrivateConfig(t *testing.T) {
	path := permissionFixture(t, 0700, 0600)
	assert.Empty(t, PermissionWarnings(path))
}

func TestPermissionWarningsWorldWritableFile(t *testing.T) {
	path := permissionFixture(t, 0700, 0666)
	assert.Len(t, PermissionWarnings(path), 1)
}

func TestPermissionWarningsWorldWritableFileAndDir(t *testing.T) {
	path := permissionFixture(t, 0777, 0666)
	assert.Len(t, PermissionWarnings(path), 2)
}

func TestPermissionWarningsMissingFileInWorldWritableDir(t *testing.T) {
	path := permissionFixture(t, 0777, 0600)
	missing := filepath.Join(filepath.Dir(path), "missing")

	assert.Len(t, PermissionWarnings(missing), 1)
}
