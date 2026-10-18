package store

import (
	"fmt"
	"os"
	"slices"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
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

func usageKeyStore(u sectionUsage) string            { return u.keyStore }
func usageTitles(u sectionUsage) []string            { return u.titles }
func usageEnvPairs(u sectionUsage) []resolve.Mapping { return u.envPairs }
func usageChannels(u sectionUsage) []resolve.Mapping { return u.channels }

func AggregateStores(cfg config.Config) map[string][]string {
	byStore := utils.GroupBy(sectionUsages(cfg), usageKeyStore)

	out := make(map[string][]string, len(byStore))
	for ks, group := range byStore {
		out[ks] = utils.SortedUnique(utils.FlatMap(group, usageTitles))
	}
	return out
}

func AggregateStoreMappings(cfg config.Config) map[string][]resolve.Mapping {
	byStore := utils.GroupBy(sectionUsages(cfg), usageKeyStore)

	out := make(map[string][]resolve.Mapping, len(byStore))
	for ks, group := range byStore {
		out[ks] = storeMappings(group)
	}
	return out
}

func sectionUsages(cfg config.Config) []sectionUsage {
	var usages []sectionUsage
	for _, section := range utils.SortedKeys(cfg.Sections) {
		if u, ok := sectionUsageOf(cfg, section); ok {
			usages = append(usages, u)
		}
	}
	return usages
}

func sectionUsageOf(cfg config.Config, section string) (sectionUsage, bool) {
	res := resolve.Resolve(cfg, section, resolve.Overrides{})
	tmplTitles := templateTitlesOrWarn(res, section)
	titles := slices.Concat(res.AllTitles(), tmplTitles)
	if res.KeyStore == "" || len(titles) == 0 {
		return sectionUsage{}, false
	}

	return sectionUsage{
		keyStore: config.ExpandHome(res.KeyStore),
		titles:   titles,
		envPairs: resolve.MappingsFromMap(res.Secrets),
		channels: channelMappings(res, tmplTitles),
	}, true
}

func templateTitlesOrWarn(res resolve.Resolved, section string) []string {
	titles, err := res.TemplateTitles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipping templates for section %q: %v\n", section, err)
	}
	return titles
}

func channelMappings(res resolve.Resolved, tmplTitles []string) []resolve.Mapping {
	return slices.Concat(
		withChannel(res.Stdin, ChannelStdin),
		withChannel(res.Files, ChannelFile),
		withChannel(tmplTitles, ChannelTemplate),
		withChannel(nonEmptyList(res.Askpass), ChannelAskpass),
	)
}

func withChannel(names []string, channel string) []resolve.Mapping {
	return utils.Map(names, func(name string) resolve.Mapping {
		return resolve.Mapping{Name: name, Env: channel}
	})
}

func nonEmptyList(name string) []string {
	if name == "" {
		return nil
	}
	return []string{name}
}

func storeMappings(group []sectionUsage) []resolve.Mapping {
	envPairs := utils.Unique(utils.FlatMap(group, usageEnvPairs))
	hasEnv := utils.ToSet(utils.Map(envPairs, resolve.MappingName))
	channels := utils.UniqueBy(utils.FlatMap(group, usageChannels), resolve.MappingName)
	channelOnly := utils.Filter(channels, func(m resolve.Mapping) bool { return !hasEnv[m.Name] })

	pairs := slices.Concat(envPairs, channelOnly)
	resolve.SortMappings(pairs)
	return pairs
}

type StoreView struct {
	Path     string
	Exists   bool
	Mappings []resolve.Mapping
}

func BuildStoreViews(cfg config.Config) []StoreView {
	byStore := AggregateStoreMappings(cfg)
	return utils.Map(utils.SortedKeys(byStore), func(path string) StoreView {
		return StoreView{Path: path, Exists: fileExists(path), Mappings: byStore[path]}
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
