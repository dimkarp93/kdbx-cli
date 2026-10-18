package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type defaultPathCandidates struct {
	legacy  string
	primary string
}

func useXDGHome(t *testing.T) defaultPathCandidates {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))

	return defaultPathCandidates{
		legacy:  filepath.Join(home, ".config", "kdbx-cli", "default"),
		primary: filepath.Join(home, "xdg", "kdbx-cli", "default"),
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o600))
}

func TestDefaultPathWithoutFiles(t *testing.T) {
	paths := useXDGHome(t)

	assert.Equal(t, paths.primary, DefaultPath())
}

func TestDefaultPathFallsBackToTheLegacyFile(t *testing.T) {
	paths := useXDGHome(t)
	touch(t, paths.legacy)

	assert.Equal(t, paths.legacy, DefaultPath())
}

func TestDefaultPathPrefersPrimaryOverLegacy(t *testing.T) {
	paths := useXDGHome(t)
	touch(t, paths.legacy)
	touch(t, paths.primary)

	assert.Equal(t, paths.primary, DefaultPath())
}
