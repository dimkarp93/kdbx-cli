package tui

import (
	"cmp"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

const pathSeparator = string(os.PathSeparator)

func refreshPathSuggestions(ti *textinput.Model) {
	dir := suggestionDir(ti.Value())
	entries, err := os.ReadDir(config.ExpandHome(dir))
	if err != nil {
		ti.SetSuggestions(nil)
		return
	}

	ti.SetSuggestions(utils.Map(entries, func(e os.DirEntry) string { return suggestionPath(dir, e) }))
}

func suggestionDir(value string) string {
	switch {
	case strings.HasSuffix(value, pathSeparator):
		return cmp.Or(strings.TrimRight(value, pathSeparator), pathSeparator)
	case value == "":
		return "."
	default:
		return filepath.Dir(value)
	}
}

func suggestionPath(dir string, e os.DirEntry) string {
	full := filepath.Join(dir, e.Name())
	if e.IsDir() {
		return full + pathSeparator
	}
	return full
}

func deleteLastPathSegment(ti *textinput.Model) {
	ti.SetValue(parentPath(ti.Value()))
	ti.CursorEnd()
}

func parentPath(path string) string {
	trimmed := strings.TrimRight(path, pathSeparator)
	i := strings.LastIndex(trimmed, pathSeparator)
	if i < 0 {
		return ""
	}
	return trimmed[:i+1]
}
