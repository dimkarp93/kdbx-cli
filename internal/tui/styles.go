package tui

import "github.com/charmbracelet/lipgloss"

const defaultTTL = "10m"

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Bold(true)
	focusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	hintStyle  = lipgloss.NewStyle().Faint(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func sectionLabel(name string, focused bool) string {
	panic("not implemented")
}
func fieldMark(focused bool) string {
	panic("not implemented")
}
func listLine(text string, selected bool) string {
	panic("not implemented")
}
