package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

type pairEditor struct {
	name  textinput.Model
	env   textinput.Model
	index int
}

func newPairEditor() pairEditor {
	panic("not implemented")
}
func (e pairEditor) start(index int, pair resolve.Mapping) (pairEditor, tea.Cmd) {
	panic("not implemented")
}
func (e pairEditor) update(msg tea.Msg) (pairEditor, tea.Cmd) {
	panic("not implemented")
}
func (e pairEditor) switchField() (pairEditor, tea.Cmd) {
	panic("not implemented")
}
func (e pairEditor) blur() pairEditor {
	panic("not implemented")
}
func (e pairEditor) value() resolve.Mapping {
	panic("not implemented")
}
func (e pairEditor) verb() string {
	panic("not implemented")
}
func (e pairEditor) view() string {
	panic("not implemented")
}
