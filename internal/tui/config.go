package tui

import (
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
	panic("not implemented")
}
func newConfigTUI(keyStore string, pairs []resolve.Mapping, cacheConfig *config.CacheConfig) configTUI {
	panic("not implemented")
}
func (m configTUI) result() ConfigResult {
	panic("not implemented")
}
func (m configTUI) Init() tea.Cmd {
	panic("not implemented")
}
func (m configTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m configTUI) cancel() (tea.Model, tea.Cmd) {
	panic("not implemented")
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
