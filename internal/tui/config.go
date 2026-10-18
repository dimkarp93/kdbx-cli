package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

type configMode int

const (
	modeKeyStore configMode = iota
	modeList
	modeEdit
	modeCache
)

type ConfigResult struct {
	KeyStore string
	Pairs    []resolve.Mapping
	Cache    *config.CacheConfig
	Canceled bool
}
type configTUI struct {
	keyStore keyStoreField
	list     pairList
	editor   pairEditor
	cache    cacheForm
	mode     configMode
	err      string
	canceled bool
}

func RunConfig(keyStore string, pairs []resolve.Mapping, cacheConfig *config.CacheConfig) (ConfigResult, error) {
	final, err := tea.NewProgram(newConfigTUI(keyStore, pairs, cacheConfig)).Run()
	if err != nil {
		return ConfigResult{}, err
	}
	return final.(configTUI).result(), nil
}
func newConfigTUI(keyStore string, pairs []resolve.Mapping, cacheConfig *config.CacheConfig) configTUI {
	return configTUI{keyStore: newKeyStoreField(keyStore), list: pairList{pairs: pairs}, editor: newPairEditor(), cache: newCacheForm(cacheConfig), mode: modeKeyStore}
}
func (m configTUI) result() ConfigResult {
	if m.canceled {
		return ConfigResult{Canceled: true}
	}
	return ConfigResult{KeyStore: m.keyStore.value(), Pairs: m.list.pairs, Cache: m.cache.config()}
}
func (m configTUI) Init() tea.Cmd {
	return textinput.Blink
}
func (m configTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, isKey := msg.(tea.KeyMsg)
	if isKey && k.String() == "ctrl+c" {
		return m.cancel()
	}
	switch m.mode {
	case modeKeyStore:
		return m.updateKeyStore(msg)
	case modeList:
		if isKey {
			return m.updateList(k.String())
		}
	case modeEdit:
		return m.updateEdit(msg)
	case modeCache:
		return m.updateCache(msg)
	}
	return m, nil
}
func (m configTUI) cancel() (tea.Model, tea.Cmd) {
	m.canceled = true
	return m, tea.Quit
}
func (m configTUI) updateKeyStore(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			return m.cancel()
		case "enter", "ctrl+s":
			m.keyStore.input.Blur()
			m.mode = modeList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.keyStore, cmd = m.keyStore.update(msg)
	return m, cmd
}
func (m configTUI) updateList(key string) (tea.Model, tea.Cmd) {
	m.err = ""
	switch key {
	case "esc":
		return m.cancel()
	case "ctrl+s":
		return m, tea.Quit
	case "tab":
		m.mode = modeKeyStore
		return m, m.keyStore.input.Focus()
	}
	return m.listAction(key)
}
func (m configTUI) listAction(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.list = m.list.moveCursor(-1)
	case "down", "j":
		m.list = m.list.moveCursor(1)
	case "a":
		return m.startEdit(-1)
	case "e", "enter":
		if len(m.list.pairs) > 0 {
			return m.startEdit(m.list.cursor)
		}
	case "d", "delete":
		m.list = m.list.deleteSelected()
	case "c":
		m.mode = modeCache
		m.cache = m.cache.reset()
	}
	return m, nil
}
func (m configTUI) startEdit(index int) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.mode = modeEdit
	m.editor, cmd = m.editor.start(index, m.list.at(index))
	return m, cmd
}
func (m configTUI) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.mode = modeList
			m.editor = m.editor.blur()
			return m, nil
		case "tab", "shift+tab":
			m.editor, cmd = m.editor.switchField()
			return m, cmd
		case "enter":
			return m.commitEdit()
		}
	}
	m.editor, cmd = m.editor.update(msg)
	return m, cmd
}
func (m configTUI) commitEdit() (tea.Model, tea.Cmd) {
	pair := m.editor.value()
	if msg := m.editError(pair); msg != "" {
		m.err = msg
		return m, nil
	}
	m.list = m.list.upsert(m.editor.index, pair)
	m.editor = m.editor.blur()
	m.mode = modeList
	m.err = ""
	return m, nil
}
func (m configTUI) editError(pair resolve.Mapping) string {
	if pair.Name == "" || pair.Env == "" {
		return "both secret name and env are required"
	}
	if m.list.envTaken(pair.Env, m.editor.index) {
		return fmt.Sprintf("env %q is already mapped", pair.Env)
	}
	return ""
}
func (m configTUI) updateCache(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.mode = modeList
			m.cache.ttl.Blur()
			return m, nil
		case "ctrl+s":
			return m, tea.Quit
		case "space", "enter":
			if m.cache.focus == focusToggle {
				m.cache = m.cache.toggle()
				return m, nil
			}
		case "tab":
			m.cache, cmd = m.cache.switchFocus()
			return m, cmd
		}
	}
	m.cache, cmd = m.cache.update(msg)
	return m, cmd
}
func (m configTUI) View() string {
	panic("not implemented")
}
func (m configTUI) legend() string {
	panic("not implemented")
}
