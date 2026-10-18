package cmd

import (
	"fmt"
	"os"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"github.com/dimkarp93/kdbx-cli/internal/term"
	"github.com/dimkarp93/kdbx-cli/internal/tui"
)

func cmdShow(path string) {
	path = configPathOrDefault(path)
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load config", path+":", err)
		os.Exit(1)
	}

	data := tui.ShowData{
		ConfigPath:  path,
		CacheConfig: cfg.Cache,
		Views:       store.BuildStoreViews(cfg),
		Cache:       keyring.New(cfg.Cache),
	}
	if !term.IsInteractive() {
		tui.PrintStores(data)
		return
	}

	if err := tui.RunShow(data, tui.LaunchKeePassXC); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}

type launchFunc func(path string) error

type showTUI struct {
	configPath  string
	cacheConfig *config.CacheConfig
	views       []store.StoreView
	cache       keyring.Cache
	launch      launchFunc
	cursor      int
	status      string
}
