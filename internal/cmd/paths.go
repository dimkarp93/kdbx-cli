package cmd

import (
	"sort"

	"github.com/dimkarp93/install-libs/pathreport"
	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func pathEntries() []pathreport.Entry {
	path := config.DefaultPath()
	entries := []pathreport.Entry{{Name: "config", Path: path}}
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
		entries = append(entries, pathreport.Entry{
			Name: "key-store:" + name,
			Path: config.ExpandHome(section.KeyStore),
		})
	}
	return entries
}
