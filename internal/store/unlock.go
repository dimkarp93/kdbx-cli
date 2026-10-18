package store

import (
	"fmt"

	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/term"
)

func UnlockExport(keyStore string, cache keyring.Cache, prompt string, allowCache bool) (string, string, error) {
	if allowCache {
		if out, pw, ok := exportCached(keyStore, cache); ok {
			return out, pw, nil
		}
	}
	return exportPrompted(keyStore, cache, prompt)
}

func exportCached(keyStore string, cache keyring.Cache) (string, string, bool) {
	pw, ok := cache.Get(keyStore)
	if !ok {
		return "", "", false
	}

	out, err := export(keyStore, pw)
	if err != nil {
		cache.Forget(keyStore)
		return "", "", false
	}
	return out, pw, true
}

func exportPrompted(keyStore string, cache keyring.Cache, prompt string) (string, string, error) {
	pw, err := term.ReadPassword(prompt)
	if err != nil {
		return "", "", err
	}
	if pw == "" {
		return "", "", fmt.Errorf("password cannot be empty")
	}

	out, err := export(keyStore, pw)
	if err != nil {
		return "", "", fmt.Errorf("keepassxc-cli export: %w: %s", err, out)
	}

	cache.Remember(keyStore, pw)
	return out, pw, nil
}

func export(keyStore, password string) (string, error) {
	return keepass.Run(password+"\n", "export", "-q", "-f", "xml", keyStore)
}
