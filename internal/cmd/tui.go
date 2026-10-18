package cmd

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

type tuiMode int

const (
	modeKeyStore tuiMode = iota
	modeList
	modeEdit
	modeCache
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Bold(true)
	focusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	hintStyle  = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type configTUI struct {
	keyStore     textinput.Model
	name         textinput.Model
	env          textinput.Model
	pairs        []resolve.Mapping
	cursor       int
	mode         tuiMode
	editIndex    int
	editName     string
	err          string
	canceled     bool
	cacheEnabled bool
	cacheTTL     textinput.Model
	cacheFocus   int
}
