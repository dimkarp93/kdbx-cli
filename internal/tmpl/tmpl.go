package tmpl

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

var placeholderRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

var wrapperExtensions = []string{".tmpl", ".tpl"}

func isJSONTemplate(path string) bool {
	base := strings.ToLower(path)
	for _, ext := range wrapperExtensions {
		base = strings.TrimSuffix(base, ext)
	}
	return strings.HasSuffix(base, ".json")
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read template %s: %w", path, err)
	}
	config.WarnUnsafePermissions(path)

	r := &renderer{values: values, jsonEscape: isJSONTemplate(path)}
	rendered := placeholderRe.ReplaceAllFunc(data, r.replace)

	if len(r.missing) > 0 {
		return nil, fmt.Errorf("template %s: no value for %s", path, strings.Join(r.missing, ", "))
	}
	return rendered, nil
}

type renderer struct {
	values     map[string]string
	jsonEscape bool
	missing    []string
}

func (r *renderer) replace(match []byte) []byte {
	name := strings.TrimSuffix(strings.TrimPrefix(string(match), "{{"), "}}")
	val, ok := r.values[name]
	if !ok {
		r.missing = append(r.missing, name)
		return match
	}

	if r.jsonEscape {
		return jsonStringBody(val)
	}
	return []byte(val)
}

func jsonStringBody(val string) []byte {
	encoded, _ := json.Marshal(val)
	return encoded[1 : len(encoded)-1]
}
