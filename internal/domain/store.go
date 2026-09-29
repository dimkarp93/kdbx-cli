package domain

import (
	"fmt"
	"os"
	"sort"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/term"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func AggregateStores(cfg config.Config) map[string][]string {
	sets := map[string]map[string]bool{}
	for section := range cfg.Sections {
		res := Resolve(cfg, section, Overrides{})
		titles := res.AllTitles()
		tmplTitles, err := res.TemplateTitles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping templates for section %q: %v\n", section, err)
		} else {
			titles = append(titles, tmplTitles...)
		}
		if res.KeyStore == "" || len(titles) == 0 {
			continue
		}
		ks := config.ExpandHome(res.KeyStore)
		if sets[ks] == nil {
			sets[ks] = map[string]bool{}
		}
		for _, name := range titles {
			sets[ks][name] = true
		}
	}
	out := make(map[string][]string, len(sets))
	for ks, set := range sets {
		titles := make([]string, 0, len(set))
		for t := range set {
			titles = append(titles, t)
		}
		sort.Strings(titles)
		out[ks] = titles
	}
	return out
}

func AggregateStoreMappings(cfg config.Config) map[string][]Mapping {
	envPairs := map[string]map[Mapping]bool{}
	channels := map[string]map[string]string{}
	sections := make([]string, 0, len(cfg.Sections))
	for section := range cfg.Sections {
		sections = append(sections, section)
	}
	sort.Strings(sections)
	for _, section := range sections {
		res := Resolve(cfg, section, Overrides{})
		tmplTitles, err := res.TemplateTitles()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping templates for section %q: %v\n", section, err)
			tmplTitles = nil
		}
		if res.KeyStore == "" || (len(res.AllTitles()) == 0 && len(tmplTitles) == 0) {
			continue
		}
		ks := config.ExpandHome(res.KeyStore)
		if envPairs[ks] == nil {
			envPairs[ks] = map[Mapping]bool{}
			channels[ks] = map[string]string{}
		}
		for env, name := range res.Secrets {
			envPairs[ks][Mapping{Name: name, Env: env}] = true
		}
		addChannel := func(name, channel string) {
			if _, ok := channels[ks][name]; !ok {
				channels[ks][name] = channel
			}
		}
		for _, name := range res.Stdin {
			addChannel(name, ChannelStdin)
		}
		for _, name := range res.Files {
			addChannel(name, ChannelFile)
		}
		for _, name := range tmplTitles {
			addChannel(name, ChannelTemplate)
		}
		if res.Askpass != "" {
			addChannel(res.Askpass, ChannelAskpass)
		}
	}
	out := make(map[string][]Mapping, len(envPairs))
	for ks, set := range envPairs {
		hasEnv := map[string]bool{}
		pairs := make([]Mapping, 0, len(set)+len(channels[ks]))
		for p := range set {
			hasEnv[p.Name] = true
			pairs = append(pairs, p)
		}
		for name, channel := range channels[ks] {
			if !hasEnv[name] {
				pairs = append(pairs, Mapping{Name: name, Env: channel})
			}
		}
		SortMappings(pairs)
		out[ks] = pairs
	}
	return out
}

type storeState struct {
	path     string
	exists   bool
	password string
	missing  []string
}

func gatherMissing(byStore map[string][]string, cache keyring.Cache) []storeState {
	stores := make([]string, 0, len(byStore))
	for ks := range byStore {
		stores = append(stores, ks)
	}
	sort.Strings(stores)

	var states []storeState
	for _, ks := range stores {
		titles := byStore[ks]
		if len(titles) == 0 {
			continue
		}
		s := storeState{path: ks}
		if !fileExists(ks) {
			s.missing = append([]string{}, titles...)
			states = append(states, s)
			continue
		}
		s.exists = true
		out, pw, err := UnlockExport(ks, cache, fmt.Sprintf("Enter password for %s: ", ks))
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", ks, err)
			continue
		}
		s.password = pw
		entries, _ := keepass.ParseSecrets(out)
		for _, t := range titles {
			if _, e := keepass.LookupSecret(entries, t); e != nil {
				s.missing = append(s.missing, t)
			}
		}
		states = append(states, s)
	}
	return states
}

func ReconcileStores(byStore map[string][]string, assumeYes bool, cache keyring.Cache) {
	keepass.CheckEngine()
	states := gatherMissing(byStore, cache)

	total := 0
	for _, s := range states {
		total += len(s.missing)
	}
	if total == 0 {
		fmt.Println("All secrets are present in the key-store(s).")
		return
	}

	fmt.Println("Missing secrets:")
	for _, s := range states {
		if len(s.missing) == 0 {
			continue
		}
		suffix := ""
		if !s.exists {
			suffix = "  (key-store does not exist)"
		}
		fmt.Printf("  %s%s\n", s.path, suffix)
		for _, t := range s.missing {
			fmt.Printf("    - %s\n", t)
		}
	}

	for i := range states {
		s := &states[i]
		if len(s.missing) == 0 {
			continue
		}
		if !s.exists {
			if !term.Confirm(fmt.Sprintf("Create key-store %s and add %d empty secret(s)?", s.path, len(s.missing)), assumeYes) {
				fmt.Println("Skipped", s.path)
				continue
			}
			pw := term.ReadPassword("Set a password for the new key-store " + s.path + ": ")
			if err := keepass.CreateStore(s.path, pw); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			s.password = pw
			cache.Remember(s.path, pw)
			fmt.Println("Created key-store", s.path)
		} else if !term.Confirm(fmt.Sprintf("Add %d empty secret(s) to %s?", len(s.missing), s.path), assumeYes) {
			fmt.Println("Skipped", s.path)
			continue
		}
		for _, t := range s.missing {
			if err := keepass.AddEmptySecret(s.path, s.password, t); err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			fmt.Printf("  + %s → %s\n", t, s.path)
		}
	}
}

type StoreView struct {
	Path     string
	Exists   bool
	Mappings []Mapping
}

func BuildStoreViews(cfg config.Config) []StoreView {
	byStore := AggregateStoreMappings(cfg)
	paths := make([]string, 0, len(byStore))
	for p := range byStore {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	views := make([]StoreView, 0, len(paths))
	for _, p := range paths {
		views = append(views, StoreView{
			Path:     p,
			Exists:   fileExists(p),
			Mappings: byStore[p],
		})
	}
	return views
}
