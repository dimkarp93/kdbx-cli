package tui

import (
	"strings"

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
	return pairEditor{name: newInput("SECRET_TITLE", ""), env: newInput("ENV_VAR", ""), index: -1}
}

func (e pairEditor) start(index int, pair resolve.Mapping) (pairEditor, tea.Cmd) {
	e.index = index
	e.name.SetValue(pair.Name)
	e.env.SetValue(pair.Env)
	e.name.CursorEnd()
	e.env.CursorEnd()
	e.env.Blur()

	return e, e.name.Focus()
}

func (e pairEditor) update(msg tea.Msg) (pairEditor, tea.Cmd) {
	var cmd tea.Cmd
	if e.name.Focused() {
		e.name, cmd = e.name.Update(msg)
	} else {
		e.env, cmd = e.env.Update(msg)
	}
	return e, cmd
}

func (e pairEditor) switchField() (pairEditor, tea.Cmd) {
	if e.name.Focused() {
		e.name.Blur()
		return e, e.env.Focus()
	}
	e.env.Blur()
	return e, e.name.Focus()
}

func (e pairEditor) blur() pairEditor {
	e.name.Blur()
	e.env.Blur()
	return e
}

func (e pairEditor) value() resolve.Mapping {
	return resolve.Mapping{Name: strings.TrimSpace(e.name.Value()), Env: strings.TrimSpace(e.env.Value())}
}

func (e pairEditor) verb() string {
	if e.index < 0 {
		return "Add"
	}
	return "Edit"
}

func (e pairEditor) view() string {
	return "\n" + labelStyle.Render("  "+e.verb()+" mapping") + "\n" +
		fieldMark(e.name.Focused()) + "name (Title in .kdbx): " + e.name.View() + "\n" +
		fieldMark(e.env.Focused()) + "env variable:          " + e.env.View() + "\n"
}
