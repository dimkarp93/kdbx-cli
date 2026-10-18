package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

func TestLoadRunConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent")

	gotPath, cfg, err := loadRunConfig(path)

	t.Run("succeeds", func(t *testing.T) {
		require.NoError(t, err)
	})
	t.Run("returns requested path", func(t *testing.T) {
		assert.Equal(t, path, gotPath)
	})
	t.Run("empty config", func(t *testing.T) {
		assert.Empty(t, cfg.Sections)
	})
}

func TestLoadRunConfigInvalidFile(t *testing.T) {
	path := writeTestFile(t, filepath.Join(t.TempDir(), "bad"), "{not json")
	_, _, err := loadRunConfig(path)
	assert.Error(t, err)
}

func TestRequestedTitlesMergesChannels(t *testing.T) {
	tmpl := writeTestFile(t, filepath.Join(t.TempDir(), "t.txt"), "{{C}} and {{A}}")
	res := resolve.Resolved{
		Secrets:   map[string]string{"X": "A"},
		Stdin:     []string{"B"},
		Templates: []config.Template{{Name: "t", Path: tmpl}},
	}

	got, err := requestedTitles(res)

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"A", "B", "C"}, got)
}

func TestRequestedTitlesTemplateError(t *testing.T) {
	res := resolve.Resolved{Templates: []config.Template{{Name: "t", Path: filepath.Join(t.TempDir(), "absent")}}}

	_, err := requestedTitles(res)
	assert.Error(t, err)
}

func TestCmdLoadRejectsDryRun(t *testing.T) {
	assert.Equal(t, 2, cmdLoad(runFlags{dryRun: true}))
}
