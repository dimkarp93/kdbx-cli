package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"os"
)

const pathSeparator = string(os.PathSeparator)

func refreshPathSuggestions(ti *textinput.Model) {
	panic("not implemented")
}
func suggestionDir(value string) string {
	panic("not implemented")
}
func suggestionPath(dir string, e os.DirEntry) string {
	panic("not implemented")
}
func deleteLastPathSegment(ti *textinput.Model) {
	panic("not implemented")
}
func parentPath(path string) string {
	panic("not implemented")
}
