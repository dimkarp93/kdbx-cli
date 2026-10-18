package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
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

		inverted, err := invertSecrets(sectionName, oldSecrets)
		if err != nil {
			return err
		}
		section["secrets"] = inverted
	}
	return nil
}

func invertSecrets(sectionName string, nameToEnv map[string]any) (map[string]any, error) {
	inverted := make(map[string]any, len(nameToEnv))
	for _, name := range slices.Sorted(maps.Keys(nameToEnv)) {
		env, ok := nameToEnv[name].(string)
		if !ok {
			return nil, fmt.Errorf("section %q: secret %q has a non-string env value", sectionName, name)
		}
		if prev, dup := inverted[env]; dup {
			return nil, fmt.Errorf("section %q: env %q is mapped from both %q and %q", sectionName, env, prev, name)
		}
		inverted[env] = name
	}
	return inverted, nil
}

func Migrate(path string, from, to int) error {
	if from < 0 || from > to || to > len(steps) {
		return fmt.Errorf("unsupported migration range %d -> %d (supported: 0 -> %d)", from, to, len(steps))
	}

	raw, err := readRawConfig(path)
	if err != nil {
		return err
	}

	version := rawVersion(raw)
	if version == to {
		return ErrAlreadyMigrated
	}
	if version != from {
		return fmt.Errorf("config %s has version %d, but --from is %d", path, version, from)
	}

	if err := applySteps(raw, from, to); err != nil {
		return err
	}
	return writeRawConfig(path, raw)
}

func readRawConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return raw, nil
}

func rawVersion(raw map[string]any) int {
	v, ok := raw["version"].(float64)
	if !ok {
		return 0
	}
	return int(v)
}

func applySteps(raw map[string]any, from, to int) error {
	for v := from; v < to; v++ {
		if err := steps[v](raw); err != nil {
			return err
		}
		raw["version"] = v + 1
	}
	return nil
}

func writeRawConfig(path string, raw map[string]any) error {
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0600)
}
