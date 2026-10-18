package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type keyStoreField struct{ input textinput.Model }

func newKeyStoreField(keyStore string) keyStoreField {
	panic("not implemented")
}
func (f keyStoreField) update(msg tea.Msg) (keyStoreField, tea.Cmd) {
	panic("not implemented")
}
func (f keyStoreField) expandHome() (keyStoreField, bool) {
	panic("not implemented")
}
func (f keyStoreField) value() string {
	panic("not implemented")
}
func newInput(placeholder, value string) textinput.Model {
	panic("not implemented")
}
