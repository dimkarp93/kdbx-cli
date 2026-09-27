package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var templatePlaceholderRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

var templateWrapperExtensions = []string{".tmpl", ".tpl"}

func isJSONTemplate(path string) bool {
	base := strings.ToLower(path)
	for _, ext := range templateWrapperExtensions {
		base = strings.TrimSuffix(base, ext)
	}
	return strings.HasSuffix(base, ".json")
}

func TemplateTitles(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read template %s: %w", path, err)
	}
	seen := map[string]bool{}
	var out []string
	for _, m := range templatePlaceholderRe.FindAllSubmatch(data, -1) {
		name := string(m[1])
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}

func RenderTemplate(path string, values map[string]string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read template %s: %w", path, err)
	}
	jsonEscape := isJSONTemplate(path)

	var missing []string
	rendered := templatePlaceholderRe.ReplaceAllFunc(data, func(match []byte) []byte {
		name := strings.TrimSuffix(strings.TrimPrefix(string(match), "{{"), "}}")
		val, ok := values[name]
		if !ok {
			missing = append(missing, name)
			return match
		}
		if jsonEscape {
			encoded, _ := json.Marshal(val)
			return encoded[1 : len(encoded)-1]
		}
		return []byte(val)
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("template %s: no value for %s", path, strings.Join(missing, ", "))
	}
	return rendered, nil
}

func (r Resolved) TemplateTitles() ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, t := range r.Templates {
		names, err := TemplateTitles(t.Path)
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out, nil
}
