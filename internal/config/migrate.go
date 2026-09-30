package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	if from < 0 || from > to || to > len(steps) {
		return fmt.Errorf("unsupported migration range %d -> %d (supported: 0 -> %d)", from, to, len(steps))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid config %s: %w", path, err)
	}
	version := 0
	if v, ok := raw["version"].(float64); ok {
		version = int(v)
	}
	if version == to {
		return ErrAlreadyMigrated
	}
	if version != from {
		return fmt.Errorf("config %s has version %d, but --from is %d", path, version, from)
	}
	for v := from; v < to; v++ {
		if err := steps[v](raw); err != nil {
			return err
		}
		raw["version"] = v + 1
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0600)
}
