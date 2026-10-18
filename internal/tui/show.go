package tui

import (
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"time"
)

type LaunchFunc func(path string) error

func LaunchKeePassXC(path string) error {
	panic("not implemented")
}

type ShowData struct {
	ConfigPath  string
	CacheConfig *config.CacheConfig
	Views       []store.StoreView
	Cache       keyring.Cache
}

func (d ShowData) cachingEnabled() bool {
	panic("not implemented")
}
func (d ShowData) cachingState() string {
	panic("not implemented")
}
func (d ShowData) storeDetails(v store.StoreView) []string {
	panic("not implemented")
}
func mappingLine(mp resolve.Mapping) string {
	panic("not implemented")
}
func cacheDetails(cache keyring.Cache, keyStore string) []string {
	panic("not implemented")
}
func missingMark(exists bool) string {
	panic("not implemented")
}
func PrintStores(d ShowData) {
	panic("not implemented")
}
func GrantedLine(cache keyring.Cache, keyStore string) string {
	panic("not implemented")
}
func FormatDuration(d time.Duration) string {
	panic("not implemented")
}
func RunShow(d ShowData, launch LaunchFunc) error {
	panic("not implemented")
}
