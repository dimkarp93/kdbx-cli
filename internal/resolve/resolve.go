package resolve

import (
	"cmp"
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"maps"
	"slices"
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
	layers := slices.Concat(sectionLayers(cfg, tool), []config.Section{config.Section(over)})
	r := Resolved{Secrets: map[string]string{}}
	for _, layer := range layers {
		r = r.overlay(layer)
	}
	return r
}
func sectionLayers(cfg config.Config, tool string) []config.Section {
	var layers []config.Section
	if s, ok := cfg.Sections["default"]; ok {
		layers = append(layers, s)
	}
	if tool == "" || tool == "default" {
		return layers
	}
	if s, ok := cfg.Sections[tool]; ok {
		layers = append(layers, s)
	}
	return layers
}
func (r Resolved) overlay(s config.Section) Resolved {
	return Resolved{KeyStore: cmp.Or(s.KeyStore, r.KeyStore), Secrets: mergeSecrets(r.Secrets, s.Secrets), Stdin: nonEmptyOr(s.Stdin, r.Stdin), StdinKeepOpen: r.StdinKeepOpen || s.StdinKeepOpen, Files: nonEmptyOr(s.Files, r.Files), Templates: nonEmptyOr(s.Templates, r.Templates), Askpass: cmp.Or(s.Askpass, r.Askpass), AskpassReads: positiveOr(s.AskpassReads, r.AskpassReads)}
}
func mergeSecrets(base, over map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(over))
	maps.Copy(out, base)
	maps.Copy(out, over)
	return out
}
func nonEmptyOr[T any](value, fallback []T) []T {
	if len(value) > 0 {
		return value
	}
	return fallback
}
func positiveOr(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
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
