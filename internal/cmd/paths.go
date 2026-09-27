package cmd

import (
	"sort"

	"github.com/dimkarp93/install-libs/xdgpath"
	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func pathEntries() []xdgpath.Entry {
	path := config.DefaultPath()
	entries := []xdgpath.Entry{{Name: "config", Path: path}}
	cfg, err := config.Load(path)
	if err != nil {
		return entries
	}
	names := make([]string, 0, len(cfg.Sections))
	for name := range cfg.Sections {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		section := cfg.Sections[name]
		if section.KeyStore == "" {
			continue
		}
		entries = append(entries, xdgpath.Entry{
			Name: "key-store:" + name,
			Path: config.ExpandHome(section.KeyStore),
		})
	}
	return entries
}
