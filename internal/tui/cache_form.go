package tui

import (
	"cmp"
	"strings"

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
	enabled, ttl := initialCache(c)
	return cacheForm{enabled: enabled, ttl: newInput(defaultTTL, ttl)}
}

func initialCache(c *config.CacheConfig) (bool, string) {
	if c == nil || !c.Enabled {
		return false, defaultTTL
	}
	return true, cmp.Or(c.TTL, defaultTTL)
}

func (f cacheForm) reset() cacheForm {
	f.focus = focusToggle
	f.ttl.Blur()
	return f
}

func (f cacheForm) toggle() cacheForm {
	f.enabled = !f.enabled
	if !f.enabled {
		f.ttl.Blur()
	}
	return f
}

func (f cacheForm) switchFocus() (cacheForm, tea.Cmd) {
	if !f.enabled {
		return f, nil
	}
	if f.focus == focusToggle {
		f.focus = focusTTL
		return f, f.ttl.Focus()
	}
	return f.reset(), nil
}

func (f cacheForm) update(msg tea.Msg) (cacheForm, tea.Cmd) {
	if f.focus != focusTTL || !f.enabled {
		return f, nil
	}

	var cmd tea.Cmd
	f.ttl, cmd = f.ttl.Update(msg)
	return f, cmd
}

func (f cacheForm) config() *config.CacheConfig {
	if !f.enabled {
		return nil
	}
	return &config.CacheConfig{Enabled: true, TTL: cmp.Or(strings.TrimSpace(f.ttl.Value()), defaultTTL)}
}

func (f cacheForm) view(focused bool) string {
	var b strings.Builder
	b.WriteString("\n" + labelStyle.Render(sectionLabel("cache", focused)) + "\n")
	b.WriteString(listLine(cacheToggle(f.enabled), focused && f.focus == focusToggle))
	if f.enabled {
		b.WriteString(f.viewTTL(focused && f.focus == focusTTL))
	}
	return b.String()
}

func cacheToggle(enabled bool) string {
	if enabled {
		return "[x] keyring enabled"
	}
	return "[ ] keyring disabled"
}

func (f cacheForm) viewTTL(focused bool) string {
	ttlLine := "TTL: " + f.ttl.View()
	if focused {
		return focusStyle.Render("  > ") + ttlLine + "\n"
	}
	return "    " + ttlLine + "\n"
}
