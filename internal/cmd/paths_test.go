package cmd

import (
	"path/filepath"
	"testing"

	"github.com/dimkarp93/install-libs/pathreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func useTempHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	return home
}

func saveDefaultConfig(t *testing.T, cfg config.Config) string {
	t.Helper()
	path := config.DefaultPath()
	require.NoError(t, config.Save(path, cfg))
	return path
}

func TestPathEntriesListsKeyStores(t *testing.T) {
	home := useTempHome(t)
	path := saveDefaultConfig(t, config.Config{Sections: map[string]config.Section{
		"work":    {KeyStore: "~/work.kdbx"},
		"default": {KeyStore: filepath.Join(home, "default.kdbx")},
		"empty":   {},
	}})

	assert.Equal(t, []pathreport.Entry{
		{Name: "config", Path: path},
		{Name: "key-store:default", Path: filepath.Join(home, "default.kdbx")},
		{Name: "key-store:work", Path: filepath.Join(home, "work.kdbx")},
	}, pathEntries())
}

func TestPathEntriesWithoutAConfig(t *testing.T) {
	useTempHome(t)

	assert.Equal(t, []pathreport.Entry{{Name: "config", Path: config.DefaultPath()}}, pathEntries())
}
