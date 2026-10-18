package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"github.com/dimkarp93/kdbx-cli/internal/term"
)

func cmdConfig(path string, assumeYes bool) {
	if path == "" {
		path = config.DefaultPath()
	}

	cfg := config.Config{}
	if c, err := config.Load(path); err == nil {
		cfg = c
	}
	if cfg.Sections == nil {
		cfg.Sections = map[string]config.Section{}
	}
	def := cfg.Sections["default"]
	pairs := resolve.MappingsFromMap(def.Secrets)

	var keyStore string
	var canceled bool
	var newCache *config.CacheConfig
	if term.IsInteractive() {
		ks, ps, cc, c, err := runConfigTUI(def.KeyStore, pairs, cfg.Cache)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tui error:", err)
			os.Exit(1)
		}
		keyStore, pairs, newCache, canceled = ks, ps, cc, c
	} else {
		keyStore, pairs = configFallback(def.KeyStore, pairs)
		newCache = cfg.Cache
	}

	if canceled {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		os.Exit(1)
	}

	def.KeyStore = keyStore
	def.Secrets = resolve.MappingsToMap(pairs)
	cfg.Sections["default"] = def
	cfg.Cache = newCache

	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "cannot write config:", err)
		os.Exit(1)
	}
	fmt.Println("Config saved to", path)

	if keyStore != "" && len(pairs) > 0 {
		fmt.Println()
		store.ReconcileStores(map[string][]string{config.ExpandHome(keyStore): titlesOf(pairs)}, assumeYes, keyring.New(cfg.Cache))
	}
}

func cmdCheck(path string, assumeYes bool) {
	if path == "" {
		path = config.DefaultPath()
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load config", path+":", err)
		os.Exit(1)
	}
	byStore := store.AggregateStores(cfg)
	if len(byStore) == 0 {
		fmt.Println("No secrets configured in", path)
		return
	}
	store.ReconcileStores(byStore, assumeYes, keyring.New(cfg.Cache))
}

func cmdForget(path string) {
	n, err := keyring.ForgetAll()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot clear cached passwords:", err)
		os.Exit(1)
	}
	if n < 0 {
		fmt.Println("Forgot all cached key-store passwords")
		return
	}
	fmt.Printf("Forgot %d cached key-store password(s)\n", n)
}

func titlesOf(pairs []resolve.Mapping) []string {
	titles := make([]string, 0, len(pairs))
	for _, p := range pairs {
		titles = append(titles, p.Name)
	}
	sort.Strings(titles)
	return titles
}

func configFallback(keyStore string, pairs []resolve.Mapping) (string, []resolve.Mapping) {
	fmt.Println("Configure the default section (Enter to keep the current value).")
	fmt.Println()

	fmt.Println("  # full path to the .kdbx key-store")
	keyStore = term.ReadWithPrefill("  key-store", keyStore)
	fmt.Println()

	fmt.Println("  # secrets mapping: name:env,name:env (name = entry Title in the .kdbx)")
	spec := term.ReadWithPrefill("  secrets", formatSecrets(pairs))
	m := map[string]string{}
	if err := mergeSecretsFlag(m, spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return keyStore, resolve.MappingsFromMap(m)
}

func formatSecrets(pairs []resolve.Mapping) string {
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, p.Name+":"+p.Env)
	}
	return strings.Join(parts, ",")
}
