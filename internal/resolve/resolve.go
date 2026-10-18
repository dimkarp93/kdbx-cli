package resolve

import (
	"github.com/dimkarp93/kdbx-cli/internal/config"
)

type Resolved struct {
	KeyStore      string
	Secrets       map[string]string
	Stdin         []string
	StdinKeepOpen bool
	Files         []string
	Templates     []config.Template
	Askpass       string
	AskpassReads  int
}
type Overrides struct {
	KeyStore      string
	Secrets       map[string]string
	Stdin         []string
	StdinKeepOpen bool
	Files         []string
	Templates     []config.Template
	Askpass       string
	AskpassReads  int
}

func Resolve(cfg config.Config, tool string, over Overrides) Resolved {
	panic("not implemented")
}
func sectionLayers(cfg config.Config, tool string) []config.Section {
	panic("not implemented")
}
func (r Resolved) overlay(s config.Section) Resolved {
	panic("not implemented")
}
func mergeSecrets(base, over map[string]string) map[string]string {
	panic("not implemented")
}
func nonEmptyOr[T any](value, fallback []T) []T {
	panic("not implemented")
}
func positiveOr(value, fallback int) int {
	panic("not implemented")
}
func (r Resolved) AllTitles() []string {
	panic("not implemented")
}
func isNonEmpty(s string) bool {
	panic("not implemented")
}
func (r Resolved) TemplateTitles() ([]string, error) {
	panic("not implemented")
}
func ConfiguredTitles(cfg config.Config, tool string) []string {
	panic("not implemented")
}
