package config

import (
	"errors"
	"fmt"
	"sort"
)

var ErrAlreadyMigrated = errors.New("config is already at the target version")

var steps = []func(raw map[string]any) error{migrateV0ToV1}

func migrateV0ToV1(raw map[string]any) error {
	sections, _ := raw["sections"].(map[string]any)
	for sectionName, v := range sections {
		section, ok := v.(map[string]any)
		if !ok {
			continue
		}
		oldSecrets, ok := section["secrets"].(map[string]any)
		if !ok {
			continue
		}
		names := make([]string, 0, len(oldSecrets))
		for name := range oldSecrets {
			names = append(names, name)
		}
		sort.Strings(names)
		inverted := make(map[string]any, len(oldSecrets))
		for _, name := range names {
			env, ok := oldSecrets[name].(string)
			if !ok {
				return fmt.Errorf("section %q: secret %q has a non-string env value", sectionName, name)
			}
			if prev, dup := inverted[env]; dup {
				return fmt.Errorf("section %q: env %q is mapped from both %q and %q", sectionName, env, prev, name)
			}
			inverted[env] = name
		}
		section["secrets"] = inverted
	}
	return nil
}

func Migrate(path string, from, to int) error {
	panic("not implemented")
}
