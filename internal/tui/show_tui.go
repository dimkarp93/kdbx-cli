package tui

import (
	"strings"

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
	return showTUI{data: d, launch: launch}
}

func (m showTUI) Init() tea.Cmd { return nil }

func (m showTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch k.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		return m.moveCursor(-1), nil
	case "down", "j":
		return m.moveCursor(1), nil
	case "enter":
		return m.openSelected(), nil
	}
	return m, nil
}

func (m showTUI) moveCursor(delta int) showTUI {
	next := m.cursor + delta
	if next < 0 || next >= len(m.data.Views) {
		return m
	}

	m.cursor = next
	m.status = ""
	return m
}

func (m showTUI) openSelected() showTUI {
	if len(m.data.Views) == 0 {
		return m
	}

	m.status = m.open(m.data.Views[m.cursor])
	return m
}

func (m showTUI) open(v store.StoreView) string {
	if !v.Exists {
		return "✗ file does not exist: " + v.Path
	}
	if err := m.launch(v.Path); err != nil {
		return "✗ " + err.Error()
	}
	return "→ opened in KeePassXC: " + v.Path
}

func (m showTUI) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("kdbx-cli — config view"))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("  config: ") + m.data.ConfigPath + "\n")
	b.WriteString(hintStyle.Render("  caching: "+m.data.cachingState()) + "\n")
	b.WriteString("\n")

	b.WriteString(m.viewStores())

	if m.status != "" {
		b.WriteString("\n  " + m.status + "\n")
	}
	b.WriteString("\n" + hintStyle.Render("  ↑/↓ select key-store · enter open in KeePassXC · q quit") + "\n")
	return b.String()
}

func (m showTUI) viewStores() string {
	if len(m.data.Views) == 0 {
		return hintStyle.Render("  No key-stores configured.") + "\n"
	}

	var b strings.Builder
	b.WriteString(labelStyle.Render("  key-stores") + "\n")
	for i, v := range m.data.Views {
		b.WriteString(m.viewStore(v, i == m.cursor))
	}
	return b.String()
}

func (m showTUI) viewStore(v store.StoreView, selected bool) string {
	var b strings.Builder
	b.WriteString(storePathLine(v, selected))

	for _, line := range m.data.storeDetails(v) {
		b.WriteString(hintStyle.Render("        "+line) + "\n")
	}
	return b.String()
}

func storePathLine(v store.StoreView, selected bool) string {
	missing := styledMissingMark(v.Exists)
	if selected {
		return focusStyle.Render("  > "+v.Path) + missing + "\n"
	}
	return "    " + v.Path + missing + "\n"
}

func styledMissingMark(exists bool) string {
	if exists {
		return ""
	}
	return hintStyle.Render(missingMark(exists))
}
