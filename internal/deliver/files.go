package deliver

import (
	"fmt"
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/secretpipe"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
	"os"
	"slices"
	"strings"
)

const firstExtraFD = 3

type fileContent struct {
	name    string
	content string
}
type Files []fileContent

func RenderFiles(names []string, templates []config.Template, values map[string]string) (Files, error) {
	panic("not implemented")
}
func renderTemplates(templates []config.Template, values map[string]string) ([]fileContent, error) {
	panic("not implemented")
}
func (f Files) Paths() map[string]string {
	panic("not implemented")
}
func (f Files) Open(pipes *secretpipe.Set) ([]*os.File, error) {
	panic("not implemented")
}
func CloseFiles(files []*os.File) {
	panic("not implemented")
}
func Placeholder(name string) string {
	return "{{" + name + "}}"
}
func SubstitutePlaceholders(args []string, paths map[string]string) ([]string, error) {
	out := slices.Clone(args)
	for _, name := range utils.SortedKeys(paths) {
		ph := Placeholder(name)
		if !slices.ContainsFunc(out, func(a string) bool {
			return strings.Contains(a, ph)
		}) {
			return nil, fmt.Errorf("placeholder %s for secret %q not found in the command", ph, name)
		}
		out = utils.Map(out, func(a string) string {
			return strings.ReplaceAll(a, ph, paths[name])
		})
	}
	return out, nil
}
