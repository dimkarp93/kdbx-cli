package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/domain"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
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

	res := domain.Resolve(cfg, "", flags.overrides())
	if res.KeyStore == "" {
		fmt.Fprintln(os.Stderr, "no key-store configured: set it in", cfgPath, "or pass --key-store")
		return 1
	}
	titles, err := requestedTitles(res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(titles) == 0 {
		fmt.Fprintln(os.Stderr, "nothing to load: pass secrets via flags or configure the default section")
		return 1
	}

	keyStore := config.ExpandHome(res.KeyStore)
	if _, err := os.Stat(keyStore); err != nil {
		fmt.Fprintln(os.Stderr, "key-store not found:", keyStore)
		return 1
	}

	prompt := fmt.Sprintf("Enter password for %s to load %d title(s): ", keyStore, len(titles))
	out, pw, err := domain.UnlockExport(keyStore, cache, prompt, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	entries, err := keepass.ParseSecrets(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot parse key-store:", err)
		return 1
	}
	var failures []string
	for _, name := range titles {
		if _, err := keepass.LookupSecret(entries, name); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "nothing loaded:")
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, "  "+f)
		}
		return 1
	}

	cache.Grant(keyStore, pw, titles)
	printLoaded(keyStore, cache)
	return 0
}

func printLoaded(keyStore string, cache keyring.Cache) {
	granted := cache.Granted(keyStore)
	sort.Strings(granted)
	fmt.Printf("Cache enabled for %s\n", keyStore)
	fmt.Printf("  titles: %s\n", strings.Join(granted, ", "))
	if st := cache.Status(keyStore); st.Cached {
		fmt.Printf("  expires in %s\n", formatDuration(st.ExpiresIn))
	}
}
