package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

type keyStoreField struct {
	input textinput.Model
}

func newKeyStoreField(keyStore string) keyStoreField {
	ti := newInput("/path/to/store.kdbx", keyStore)
	ti.CursorEnd()
	ti.ShowSuggestions = true
	ti.Focus()
	refreshPathSuggestions(&ti)
	return keyStoreField{input: ti}
}

func (f keyStoreField) update(msg tea.Msg) (keyStoreField, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "alt+backspace":
			deleteLastPathSegment(&f.input)
			refreshPathSuggestions(&f.input)
			return f, nil
		case "tab":
			if expanded, ok := f.expandHome(); ok {
				return expanded, nil
			}
		}
	}

	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	refreshPathSuggestions(&f.input)
	return f, cmd
}

func (f keyStoreField) expandHome() (keyStoreField, bool) {
	v := f.input.Value()
	if !strings.HasPrefix(v, "~") {
		return f, false
	}

	f.input.SetValue(config.ExpandHome(v))
	f.input.CursorEnd()
	refreshPathSuggestions(&f.input)
	return f, true
}

func (f keyStoreField) value() string {
	return strings.TrimSpace(f.input.Value())
}

func newInput(placeholder, value string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetValue(value)
	return ti
}
