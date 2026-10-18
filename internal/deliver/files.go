package deliver

import (
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/secretpipe"
	"os"
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
	panic("not implemented")
}
func SubstitutePlaceholders(args []string, paths map[string]string) ([]string, error) {
	panic("not implemented")
}
