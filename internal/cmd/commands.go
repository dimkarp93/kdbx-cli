package cmd

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"github.com/dimkarp93/kdbx-cli/internal/term"
	"github.com/dimkarp93/kdbx-cli/internal/tui"
	"github.com/dimkarp93/utils/pkg/collections"
)

func cmdConfig(path string, assumeYes bool) {
	path = configPathOrDefault(path)
	cfg := loadOrEmptyConfig(path)

	edit, err := editDefaultSection(cfg.Sections["default"], cfg.Cache)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
	if edit.Canceled {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		os.Exit(1)
	}

	cfg = applyDefaultEdit(cfg, edit)
	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "cannot write config:", err)
		os.Exit(1)
	}
	fmt.Println("Config saved to", path)

	if edit.KeyStore != "" && len(edit.Pairs) > 0 {
		fmt.Println()
		byStore := map[string][]string{config.ExpandHome(edit.KeyStore): titlesOf(edit.Pairs)}
		store.ReconcileStores(byStore, assumeYes, keyring.New(cfg.Cache))
	}
}

func loadOrEmptyConfig(path string) config.Config {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}
	}
	return cfg
}

func editDefaultSection(def config.Section, cache *config.CacheConfig) (tui.ConfigResult, error) {
	pairs := resolve.MappingsFromMap(def.Secrets)
	if term.IsInteractive() {
		return tui.RunConfig(def.KeyStore, pairs, cache)
	}

	keyStore, pairs := configFallback(def.KeyStore, pairs)
	return tui.ConfigResult{KeyStore: keyStore, Pairs: pairs, Cache: cache}, nil
}

func applyDefaultEdit(cfg config.Config, edit tui.ConfigResult) config.Config {
	def := cfg.Sections["default"]
	def.KeyStore = edit.KeyStore
	def.Secrets = resolve.MappingsToMap(edit.Pairs)

	sections := make(map[string]config.Section, len(cfg.Sections)+1)
	maps.Copy(sections, cfg.Sections)
	sections["default"] = def

	cfg.Sections = sections
	cfg.Cache = edit.Cache
	return cfg
}

func cmdCheck(path string, assumeYes bool) {
	path = configPathOrDefault(path)
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
	return slices.Sorted(slices.Values(collections.Map(pairs, resolve.MappingName)))
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
	return strings.Join(collections.Map(pairs, func(p resolve.Mapping) string { return p.Name + ":" + p.Env }), ",")
}
