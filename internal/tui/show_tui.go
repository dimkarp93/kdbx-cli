package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dimkarp93/kdbx-cli/internal/store"
)

type showTUI struct {
	data   ShowData
	launch LaunchFunc
	cursor int
	status string
}

func newShowTUI(d ShowData, launch LaunchFunc) showTUI {
	panic("not implemented")
}
func (m showTUI) Init() tea.Cmd {
	panic("not implemented")
}
func (m showTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	panic("not implemented")
}
func (m showTUI) moveCursor(delta int) showTUI {
	panic("not implemented")
}
func (m showTUI) openSelected() showTUI {
	panic("not implemented")
}
func (m showTUI) open(v store.StoreView) string {
	panic("not implemented")
}
func (m showTUI) View() string {
	panic("not implemented")
}
func (m showTUI) viewStores() string {
	panic("not implemented")
}
func (m showTUI) viewStore(v store.StoreView, selected bool) string {
	panic("not implemented")
}
func storePathLine(v store.StoreView, selected bool) string {
	panic("not implemented")
}
func styledMissingMark(exists bool) string {
	panic("not implemented")
}
