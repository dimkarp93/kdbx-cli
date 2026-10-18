package cmd

import (
	"slices"

	"github.com/dimkarp93/install-libs/pathreport"
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

func pathEntries() []pathreport.Entry {
	path := config.DefaultPath()
	configEntry := []pathreport.Entry{{Name: "config", Path: path}}
	cfg, err := config.Load(path)
	if err != nil {
		return configEntry
	}

	return slices.Concat(configEntry, keyStoreEntries(cfg))
}

func keyStoreEntries(cfg config.Config) []pathreport.Entry {
	var entries []pathreport.Entry
	for _, name := range utils.SortedKeys(cfg.Sections) {
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
