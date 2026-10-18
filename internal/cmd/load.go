package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"github.com/dimkarp93/kdbx-cli/internal/tui"
)

func cmdLoad(flags runFlags) int {
	if flags.dryRun {
		fmt.Fprintln(os.Stderr, "load does not support --dry-run")
		return 2
	}
	keepass.CheckEngine()

	cfgPath, cfg, err := loadRunConfig(flags.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cache := keyring.New(cfg.Cache)
	if !cache.Enabled() {
		fmt.Fprintln(os.Stderr, "caching is disabled: enable the \"cached\" section in", cfgPath)
		return 1
	}

	keyStore, err := loadAndGrant(cfgPath, cfg, flags, cache)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	printLoaded(keyStore, cache)
	return 0
}

func loadAndGrant(cfgPath string, cfg config.Config, flags runFlags, cache keyring.Cache) (string, error) {
	keyStore, titles, err := loadTargets(cfgPath, cfg, flags)
	if err != nil {
		return "", err
	}
	return keyStore, grantTitles(keyStore, titles, cache)
}

func loadTargets(cfgPath string, cfg config.Config, flags runFlags) (string, []string, error) {
	res := resolve.Resolve(cfg, "", flags.overrides())
	if res.KeyStore == "" {
		return "", nil, errNoKeyStore(cfgPath)
	}

	titles, err := requestedTitles(res)
	if err != nil {
		return "", nil, err
	}
	if len(titles) == 0 {
		return "", nil, errors.New("nothing to load: pass secrets via flags or configure the default section")
	}

	keyStore, err := existingKeyStore(res.KeyStore)
	return keyStore, titles, err
}

func grantTitles(keyStore string, titles []string, cache keyring.Cache) error {
	prompt := fmt.Sprintf("Enter password for %s to load %d title(s): ", keyStore, len(titles))
	out, pw, err := store.UnlockExport(keyStore, cache, prompt, false)
	if err != nil {
		return err
	}

	entries, err := parseEntries(out)
	if err != nil {
		return err
	}
	if _, failures := lookupAll(entries, titles); len(failures) > 0 {
		return failuresError("nothing loaded:", failures)
	}

	cache.Grant(keyStore, pw, titles)
	return nil
}

func printLoaded(keyStore string, cache keyring.Cache) {
	granted := slices.Sorted(slices.Values(cache.Granted(keyStore)))
	fmt.Printf("Cache enabled for %s\n", keyStore)
	fmt.Printf("  titles: %s\n", strings.Join(granted, ", "))
	if st := cache.Status(keyStore); st.Cached {
		fmt.Printf("  expires in %s\n", tui.FormatDuration(st.ExpiresIn))
	}
}
