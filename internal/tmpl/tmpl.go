package tmpl

import (
	"fmt"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
	"os"
	"regexp"
)

var placeholderRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
var wrapperExtensions = []string{".tmpl", ".tpl"}

func isJSONTemplate(path string) bool {
	panic("not implemented")
}
func Titles(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read template %s: %w", path, err)
	}
	matches := placeholderRe.FindAllSubmatch(data, -1)
	return utils.Unique(utils.Map(matches, submatchName)), nil
}
func submatchName(match [][]byte) string {
	return string(match[1])
}
func Render(path string, values map[string]string) ([]byte, error) {
	panic("not implemented")
}

type renderer struct {
	values     map[string]string
	jsonEscape bool
	missing    []string
}

func (r *renderer) replace(match []byte) []byte {
	panic("not implemented")
}
func jsonStringBody(val string) []byte {
	panic("not implemented")
}
