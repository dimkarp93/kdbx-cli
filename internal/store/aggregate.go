package store

import (
	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

const (
	ChannelStdin    = "(stdin)"
	ChannelFile     = "(file)"
	ChannelTemplate = "(template)"
	ChannelAskpass  = "(askpass)"
)

type sectionUsage struct {
	keyStore string
	titles   []string
	envPairs []resolve.Mapping
	channels []resolve.Mapping
}

func usageKeyStore(u sectionUsage) string {
	panic("not implemented")
}
func usageTitles(u sectionUsage) []string {
	panic("not implemented")
}
func usageEnvPairs(u sectionUsage) []resolve.Mapping {
	panic("not implemented")
}
func usageChannels(u sectionUsage) []resolve.Mapping {
	panic("not implemented")
}
func AggregateStores(cfg config.Config) map[string][]string {
	panic("not implemented")
}
func AggregateStoreMappings(cfg config.Config) map[string][]resolve.Mapping {
	panic("not implemented")
}
func sectionUsages(cfg config.Config) []sectionUsage {
	panic("not implemented")
}
func sectionUsageOf(cfg config.Config, section string) (sectionUsage, bool) {
	panic("not implemented")
}
func templateTitlesOrWarn(res resolve.Resolved, section string) []string {
	panic("not implemented")
}
func channelMappings(res resolve.Resolved, tmplTitles []string) []resolve.Mapping {
	panic("not implemented")
}
func withChannel(names []string, channel string) []resolve.Mapping {
	panic("not implemented")
}
func nonEmptyList(name string) []string {
	panic("not implemented")
}
func storeMappings(group []sectionUsage) []resolve.Mapping {
	panic("not implemented")
}

type StoreView struct {
	Path     string
	Exists   bool
	Mappings []resolve.Mapping
}

func BuildStoreViews(cfg config.Config) []StoreView {
	panic("not implemented")
}
func fileExists(path string) bool {
	panic("not implemented")
}
