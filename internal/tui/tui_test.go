package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/store"
)

func writeTestFile(t *testing.T, path, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestRefreshPathSuggestions(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "store.kdbx"), "")
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0755))

	ti := textinput.New()
	ti.SetValue(dir + string(os.PathSeparator))

	refreshPathSuggestions(&ti)

	assert.Subset(t, ti.AvailableSuggestions(), []string{
		filepath.Join(dir, "store.kdbx"),
		filepath.Join(dir, "sub") + string(os.PathSeparator),
	})
}

func TestDeleteLastPathSegment(t *testing.T) {
	cases := map[string]string{
		"/home/dima/vault/store.kdbx": "/home/dima/vault/",
		"/home/dima/vault/":           "/home/dima/",
		"/home/":                      "/",
		"/":                           "",
		"relative/path":               "relative/",
		"single":                      "",
	}

	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			ti := textinput.New()
			ti.SetValue(in)
			deleteLastPathSegment(&ti)
			assert.Equal(t, want, ti.Value())
		})
	}
}

func commitNewPair(pairs []resolve.Mapping, name, env string) configTUI {
	m := newConfigTUI("", pairs, nil)
	m.mode = modeEdit
	m.editor.index = -1
	m.editor.name.SetValue(name)
	m.editor.env.SetValue(env)

	res, _ := m.commitEdit()
	return res.(configTUI)
}

func TestCommitEditAddsSorted(t *testing.T) {
	m := commitNewPair([]resolve.Mapping{{Name: "B", Env: "2"}}, "A", "1")

	t.Run("sorted by name", func(t *testing.T) {
		assert.Equal(t, []resolve.Mapping{{Name: "A", Env: "1"}, {Name: "B", Env: "2"}}, m.list.pairs)
	})
	t.Run("cursor on committed pair", func(t *testing.T) {
		assert.Equal(t, 0, m.list.cursor)
	})
}

func TestCommitEditRejectsDuplicateEnv(t *testing.T) {
	m := commitNewPair([]resolve.Mapping{{Name: "B", Env: "2"}}, "C", "2")

	t.Run("reports error", func(t *testing.T) {
		assert.NotEmpty(t, m.err)
	})
	t.Run("stays in edit mode", func(t *testing.T) {
		assert.Equal(t, modeEdit, m.mode)
	})
	t.Run("keeps pairs", func(t *testing.T) {
		assert.Equal(t, []resolve.Mapping{{Name: "B", Env: "2"}}, m.list.pairs)
	})
}

func TestCommitEditAllowsSameSecretWithAnotherEnv(t *testing.T) {
	m := commitNewPair([]resolve.Mapping{{Name: "B", Env: "2"}}, "B", "9")

	t.Run("no error", func(t *testing.T) {
		assert.Empty(t, m.err)
	})
	t.Run("both pairs kept", func(t *testing.T) {
		assert.Len(t, m.list.pairs, 2)
	})
	t.Run("cursor on new pair", func(t *testing.T) {
		require.Less(t, m.list.cursor, len(m.list.pairs))
		assert.Equal(t, resolve.Mapping{Name: "B", Env: "9"}, m.list.pairs[m.list.cursor])
	})
}

func TestCommitEditRequiresBothFields(t *testing.T) {
	m := commitNewPair(nil, "", "X")

	t.Run("reports error", func(t *testing.T) {
		assert.NotEmpty(t, m.err)
	})
	t.Run("adds nothing", func(t *testing.T) {
		assert.Empty(t, m.list.pairs)
	})
}

func TestListDeleteAdjustsCursor(t *testing.T) {
	m := newConfigTUI("", []resolve.Mapping{{Name: "A", Env: "1"}, {Name: "B", Env: "2"}}, nil)
	m.mode = modeList
	m.list.cursor = 1

	res, _ := m.updateList("d")
	m = res.(configTUI)

	t.Run("removes selected pair", func(t *testing.T) {
		assert.Equal(t, []resolve.Mapping{{Name: "A", Env: "1"}}, m.list.pairs)
	})
	t.Run("clamps cursor", func(t *testing.T) {
		assert.Equal(t, 0, m.list.cursor)
	})
}

type recordingLauncher struct {
	opened string
	err    error
}

func (l *recordingLauncher) launch(path string) error {
	l.opened = path
	return l.err
}

func newTwoStoreShowTUI(launch LaunchFunc) showTUI {
	views := []store.StoreView{
		{Path: "/a.kdbx", Exists: true},
		{Path: "/b.kdbx", Exists: false},
	}
	return newShowTUI(ShowData{ConfigPath: "/cfg", Views: views, Cache: keyring.New(nil)}, launch)
}

func pressKeys(m showTUI, keys ...tea.KeyType) showTUI {
	for _, key := range keys {
		res, _ := m.Update(tea.KeyMsg{Type: key})
		m = res.(showTUI)
	}
	return m
}

func TestShowTUIDownMovesCursor(t *testing.T) {
	m := pressKeys(newTwoStoreShowTUI(new(recordingLauncher).launch), tea.KeyDown)
	assert.Equal(t, 1, m.cursor)
}

func TestShowTUIUpMovesCursor(t *testing.T) {
	m := pressKeys(newTwoStoreShowTUI(new(recordingLauncher).launch), tea.KeyDown, tea.KeyUp)
	assert.Equal(t, 0, m.cursor)
}

func TestShowTUIEnterOnMissingStore(t *testing.T) {
	launcher := &recordingLauncher{}

	m := pressKeys(newTwoStoreShowTUI(launcher.launch), tea.KeyDown, tea.KeyEnter)

	t.Run("reports missing file", func(t *testing.T) {
		assert.Contains(t, m.status, "does not exist")
	})
	t.Run("does not open", func(t *testing.T) {
		assert.Empty(t, launcher.opened)
	})
}

func TestShowTUIEnterOpensExistingStore(t *testing.T) {
	launcher := &recordingLauncher{}

	m := pressKeys(newTwoStoreShowTUI(launcher.launch), tea.KeyEnter)

	t.Run("opens selected store", func(t *testing.T) {
		assert.Equal(t, "/a.kdbx", launcher.opened)
	})
	t.Run("reports opening", func(t *testing.T) {
		assert.Contains(t, m.status, "opened in KeePassXC")
	})
}

func TestShowTUIEnterReportsLaunchError(t *testing.T) {
	launcher := &recordingLauncher{err: errors.New("keepassxc GUI not found in PATH")}

	m := pressKeys(newTwoStoreShowTUI(launcher.launch), tea.KeyEnter)

	assert.Equal(t, "✗ keepassxc GUI not found in PATH", m.status)
}
