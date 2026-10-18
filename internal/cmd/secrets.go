package cmd

import (
	"os"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

func loadRunConfig(configPath string) (string, config.Config, error) {
	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}
	cfg, err := config.Load(cfgPath)
	if err == nil {
		return cfgPath, cfg, nil
	}
	if os.IsNotExist(err) {
		return cfgPath, config.Config{}, nil
	}
	return cfgPath, config.Config{}, err
}

func requestedTitles(res resolve.Resolved) ([]string, error) {
	titles := res.AllTitles()
	tmplTitles, err := res.TemplateTitles()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(titles))
	for _, name := range titles {
		seen[name] = true
	}
	for _, name := range tmplTitles {
		if !seen[name] {
			seen[name] = true
			titles = append(titles, name)
		}
	}
	return titles, nil
}

func titlesGranted(titles []string, granted map[string]bool) bool {
	for _, name := range titles {
		if !granted[name] {
			return false
		}
	}
	return true
}
