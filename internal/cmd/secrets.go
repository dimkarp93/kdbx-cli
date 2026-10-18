package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

func configPathOrDefault(path string) string {
	if path != "" {
		return path
	}
	return config.DefaultPath()
}

func loadRunConfig(configPath string) (string, config.Config, error) {
	cfgPath := configPathOrDefault(configPath)
	cfg, err := config.Load(cfgPath)
	switch {
	case err == nil:
		return cfgPath, cfg, nil
	case os.IsNotExist(err):
		return cfgPath, config.Config{}, nil
	default:
		return cfgPath, config.Config{}, err
	}
}

func requestedTitles(res resolve.Resolved) ([]string, error) {
	tmplTitles, err := res.TemplateTitles()
	if err != nil {
		return nil, err
	}
	return utils.Unique(slices.Concat(res.AllTitles(), tmplTitles)), nil
}

func errNoKeyStore(cfgPath string) error {
	return fmt.Errorf("no key-store configured: set it in %s or pass --key-store", cfgPath)
}

func existingKeyStore(keyStore string) (string, error) {
	path := config.ExpandHome(keyStore)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("key-store not found: %s", path)
	}
	return path, nil
}

func parseEntries(export string) ([]keepass.Entry, error) {
	entries, err := keepass.ParseSecrets(export)
	if err != nil {
		return nil, fmt.Errorf("cannot parse key-store: %w", err)
	}
	return entries, nil
}

func lookupAll(entries []keepass.Entry, titles []string) (map[string]string, []string) {
	values := make(map[string]string, len(titles))
	var failures []string
	for _, name := range titles {
		val, err := keepass.LookupSecret(entries, name)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		values[name] = val
	}
	return values, failures
}

func failuresError(header string, failures []string) error {
	return errors.New(header + "\n  " + strings.Join(failures, "\n  "))
}

func titlesGranted(titles []string, granted map[string]bool) bool {
	for _, name := range titles {
		if !granted[name] {
			return false
		}
	}
	return true
}
