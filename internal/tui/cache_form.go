package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dimkarp93/kdbx-cli/internal/config"
)

type cacheFocus int

const (
	focusToggle cacheFocus = iota
	focusTTL
)

type cacheForm struct {
	enabled bool
	ttl     textinput.Model
	focus   cacheFocus
}

func newCacheForm(c *config.CacheConfig) cacheForm {
	panic("not implemented")
}
func initialCache(c *config.CacheConfig) (bool, string) {
	panic("not implemented")
}
func (f cacheForm) reset() cacheForm {
	panic("not implemented")
}
func (f cacheForm) toggle() cacheForm {
	panic("not implemented")
}
func (f cacheForm) switchFocus() (cacheForm, tea.Cmd) {
	panic("not implemented")
}
func (f cacheForm) update(msg tea.Msg) (cacheForm, tea.Cmd) {
	panic("not implemented")
}
func (f cacheForm) config() *config.CacheConfig {
	panic("not implemented")
}
func (f cacheForm) view(focused bool) string {
	panic("not implemented")
}
func cacheToggle(enabled bool) string {
	panic("not implemented")
}
func (f cacheForm) viewTTL(focused bool) string {
	panic("not implemented")
}
