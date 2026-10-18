package tmpl

import (
	"regexp"
)

var placeholderRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
var wrapperExtensions = []string{".tmpl", ".tpl"}

func isJSONTemplate(path string) bool {
	panic("not implemented")
}
func Titles(path string) ([]string, error) {
	panic("not implemented")
}
func submatchName(match [][]byte) string {
	panic("not implemented")
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
