package tui

import (
	"cmp"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

type LaunchFunc func(path string) error

func LaunchKeePassXC(path string) error {
	bin, err := exec.LookPath("keepassxc")
	if err != nil {
		return fmt.Errorf("keepassxc GUI not found in PATH")
	}

	cmd := exec.Command(bin, path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

type ShowData struct {
	ConfigPath  string
	CacheConfig *config.CacheConfig
	Views       []store.StoreView
	Cache       keyring.Cache
}

func (d ShowData) cachingEnabled() bool {
	return d.CacheConfig != nil && d.CacheConfig.Enabled
}

func (d ShowData) cachingState() string {
	if !d.cachingEnabled() {
		return "disabled"
	}
	return fmt.Sprintf("enabled (TTL: %s)", cmp.Or(d.CacheConfig.TTL, defaultTTL))
}

func (d ShowData) storeDetails(v store.StoreView) []string {
	mappings := utils.Map(v.Mappings, mappingLine)
	if !d.cachingEnabled() {
		return mappings
	}
	return slices.Concat(mappings, cacheDetails(d.Cache, v.Path))
}

func mappingLine(mp resolve.Mapping) string {
	return fmt.Sprintf("%s → %s", mp.Name, mp.Env)
}

func cacheDetails(cache keyring.Cache, keyStore string) []string {
	st := cache.Status(keyStore)
	if !st.Cached {
		return []string{"[not cached]"}
	}

	expiry := fmt.Sprintf("[cached, expires in %s]", FormatDuration(st.ExpiresIn))
	if line := GrantedLine(cache, keyStore); line != "" {
		return []string{expiry, line}
	}
	return []string{expiry}
}

func missingMark(exists bool) string {
	if exists {
		return ""
	}
	return "  (missing)"
}

func PrintStores(d ShowData) {
	fmt.Println("Config:", d.ConfigPath)
	fmt.Println()
	fmt.Println("Caching:", d.cachingState())
	fmt.Println()

	if len(d.Views) == 0 {
		fmt.Println("No key-stores configured.")
		return
	}

	fmt.Println("Key-stores:")
	for _, v := range d.Views {
		fmt.Printf("  %s%s\n", v.Path, missingMark(v.Exists))
		for _, line := range d.storeDetails(v) {
			fmt.Printf("      %s\n", line)
		}
	}
}

func GrantedLine(cache keyring.Cache, keyStore string) string {
	granted := cache.Granted(keyStore)
	if len(granted) == 0 {
		return ""
	}
	return "granted via load: " + strings.Join(granted, ", ")
}

func FormatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	switch {
	case h > 0:
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func RunShow(d ShowData, launch LaunchFunc) error {
	_, err := tea.NewProgram(newShowTUI(d, launch)).Run()
	return err
}
