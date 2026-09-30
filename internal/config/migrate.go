package config

import "errors"

var ErrAlreadyMigrated = errors.New("config is already at the target version")

var steps = []func(raw map[string]any) error{migrateV0ToV1}

func migrateV0ToV1(raw map[string]any) error {
	panic("not implemented")
}

func Migrate(path string, from, to int) error {
	panic("not implemented")
}
