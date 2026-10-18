package tui

import (
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
	panic("not implemented")
}
func (m configTUI) cancel() (tea.Model, tea.Cmd) {
	m.canceled = true
	return m, tea.Quit
}
func (m configTUI) updateKeyStore(msg tea.Msg) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) updateList(key string) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) listAction(key string) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) startEdit(index int) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) commitEdit() (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) editError(pair resolve.Mapping) string {
	panic("not implemented")
}
func (m configTUI) updateCache(msg tea.Msg) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) View() string {
	panic("not implemented")
}
func (m configTUI) legend() string {
	panic("not implemented")
}
