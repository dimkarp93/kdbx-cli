package deliver

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/secretpipe"
	"github.com/dimkarp93/kdbx-cli/internal/tmpl"
	"github.com/dimkarp93/utils/pkg/collections"
)

const firstExtraFD = 3

type fileContent struct {
	name    string
	content string
}

type Files []fileContent

func RenderFiles(names []string, templates []config.Template, values map[string]string) (Files, error) {
	rendered, err := renderTemplates(templates, values)
	if err != nil {
		return nil, err
	}

	secrets := collections.Map(names, func(name string) fileContent {
		return fileContent{name: name, content: values[name]}
	})
	return slices.Concat(secrets, rendered), nil
}

func renderTemplates(templates []config.Template, values map[string]string) ([]fileContent, error) {
	out := make([]fileContent, 0, len(templates))
	for _, t := range templates {
		data, err := tmpl.Render(t.Path, values)
		if err != nil {
			return nil, err
		}
		out = append(out, fileContent{name: t.Name, content: string(data)})
	}
	return out, nil
}

func (f Files) Paths() map[string]string {
	paths := make(map[string]string, len(f))
	for i, c := range f {
		paths[c.name] = fmt.Sprintf("/dev/fd/%d", firstExtraFD+i)
	}
	return paths
}

func (f Files) Open(pipes *secretpipe.Set) ([]*os.File, error) {
	opened := make([]*os.File, 0, len(f))
	for _, c := range f {
		file, err := pipes.File(c.name, c.content)
		if err != nil {
			CloseFiles(opened)
			return nil, err
		}
		opened = append(opened, file)
	}
	return opened, nil
}

func CloseFiles(files []*os.File) {
	for _, f := range files {
		f.Close()
	}
}

func Placeholder(name string) string { return "{{" + name + "}}" }

func SubstitutePlaceholders(args []string, paths map[string]string) ([]string, error) {
	out := slices.Clone(args)
	for _, name := range collections.SortedKeys(paths) {
		ph := Placeholder(name)
		if !slices.ContainsFunc(out, func(a string) bool { return strings.Contains(a, ph) }) {
			return nil, fmt.Errorf("placeholder %s for secret %q not found in the command", ph, name)
		}
		out = collections.Map(out, func(a string) string { return strings.ReplaceAll(a, ph, paths[name]) })
	}
	return out, nil
}
