package domain

import (
	"maps"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

type Resolved struct {
	KeyStore      string
	Secrets       map[string]string
	Stdin         []string
	StdinKeepOpen bool
	Files         []string
	Askpass       string
}

type Overrides struct {
	KeyStore      string
	Secrets       map[string]string
	Stdin         []string
	StdinKeepOpen bool
	Files         []string
	Askpass       string
}

func Resolve(cfg config.Config, tool string, over Overrides) Resolved {
	r := Resolved{Secrets: map[string]string{}}
	apply := func(s config.Section) {
		if s.KeyStore != "" {
			r.KeyStore = s.KeyStore
		}
		maps.Copy(r.Secrets, s.Secrets)
		if len(s.Stdin) > 0 {
			r.Stdin = s.Stdin
		}
		if s.StdinKeepOpen {
			r.StdinKeepOpen = true
		}
		if len(s.Files) > 0 {
			r.Files = s.Files
		}
		if s.Askpass != "" {
			r.Askpass = s.Askpass
		}
	}
	if s, ok := cfg.Sections["default"]; ok {
		apply(s)
	}
	if tool != "" && tool != "default" {
		if s, ok := cfg.Sections[tool]; ok {
			apply(s)
		}
	}
	if over.KeyStore != "" {
		r.KeyStore = over.KeyStore
	}
	maps.Copy(r.Secrets, over.Secrets)
	if len(over.Stdin) > 0 {
		r.Stdin = over.Stdin
	}
	if over.StdinKeepOpen {
		r.StdinKeepOpen = true
	}
	if len(over.Files) > 0 {
		r.Files = over.Files
	}
	if over.Askpass != "" {
		r.Askpass = over.Askpass
	}
	return r
}

func (r Resolved) AllTitles() []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for name := range r.Secrets {
		add(name)
	}
	for _, name := range r.Stdin {
		add(name)
	}
	for _, name := range r.Files {
		add(name)
	}
	add(r.Askpass)
	return out
}
