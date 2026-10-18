package resolve

import (
	"cmp"
	"maps"
	"slices"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/tmpl"
	"github.com/dimkarp93/utils/pkg/collections"
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
	return Resolved{
		KeyStore:      cmp.Or(s.KeyStore, r.KeyStore),
		Secrets:       mergeSecrets(r.Secrets, s.Secrets),
		Stdin:         nonEmptyOr(s.Stdin, r.Stdin),
		StdinKeepOpen: r.StdinKeepOpen || s.StdinKeepOpen,
		Files:         nonEmptyOr(s.Files, r.Files),
		Templates:     nonEmptyOr(s.Templates, r.Templates),
		Askpass:       cmp.Or(s.Askpass, r.Askpass),
		AskpassReads:  positiveOr(s.AskpassReads, r.AskpassReads),
	}
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
	titles := slices.Concat(slices.Collect(maps.Values(r.Secrets)), r.Stdin, r.Files, []string{r.Askpass})
	return collections.Unique(collections.Filter(titles, isNonEmpty))
}

func isNonEmpty(s string) bool {
	return s != ""
}

func (r Resolved) TemplateTitles() ([]string, error) {
	perTemplate := make([][]string, 0, len(r.Templates))
	for _, t := range r.Templates {
		names, err := tmpl.Titles(t.Path)
		if err != nil {
			return nil, err
		}
		perTemplate = append(perTemplate, names)
	}
	return collections.Unique(slices.Concat(perTemplate...)), nil
}

func ConfiguredTitles(cfg config.Config, tool string) []string {
	res := Resolve(cfg, tool, Overrides{})
	tmplTitles, _ := res.TemplateTitles()
	return collections.Unique(slices.Concat(res.AllTitles(), tmplTitles))
}
