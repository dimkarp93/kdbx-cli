package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateV0ToV1InvertsSecrets(t *testing.T) {
	path := writeRaw(t, `{"sections":{"default":{"key-store":"/k.kdbx","secrets":{"Token":"GH_TOKEN","Pw":"DB_PASSWORD"}},"psql":{"secrets":{"Pw":"PGPASSWORD"}}},"cached":{"enabled":true,"ttl":"5m"}}`)

	migrateErr := Migrate(path, 0, 1)
	c, loadErr := Load(path)

	t.Run("migrates", func(t *testing.T) {
		require.NoError(t, migrateErr)
	})
	t.Run("loads", func(t *testing.T) {
		require.NoError(t, loadErr)
	})
	t.Run("default section", func(t *testing.T) {
		want := Section{KeyStore: "/k.kdbx", Secrets: map[string]string{"GH_TOKEN": "Token", "DB_PASSWORD": "Pw"}}
		assert.Equal(t, want, c.Sections["default"])
	})
	t.Run("tool section", func(t *testing.T) {
		assert.Equal(t, map[string]string{"PGPASSWORD": "Pw"}, c.Sections["psql"].Secrets)
	})
	t.Run("cache kept", func(t *testing.T) {
		assert.Equal(t, &CacheConfig{Enabled: true, TTL: "5m"}, c.Cache)
	})
}

func TestMigrateCollisionLeavesFileUntouched(t *testing.T) {
	body := `{"sections":{"default":{"secrets":{"A":"X","B":"X"}}}}`
	path := writeRaw(t, body)

	err := Migrate(path, 0, 1)

	t.Run("names colliding env", func(t *testing.T) {
		assert.ErrorContains(t, err, `"X"`)
	})
	t.Run("file unchanged", func(t *testing.T) {
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, body, string(data))
	})
}

func TestMigrateAlreadyAtTarget(t *testing.T) {
	path := writeRaw(t, `{"version":1,"sections":{}}`)
	assert.ErrorIs(t, Migrate(path, 0, 1), ErrAlreadyMigrated)
}

func TestMigrateUnsupportedRange(t *testing.T) {
	path := writeRaw(t, `{"version":1,"sections":{}}`)
	assert.Error(t, Migrate(path, 0, 2))
}

func TestMigrateFromMismatch(t *testing.T) {
	path := writeRaw(t, `{"sections":{}}`)

	err := Migrate(path, 1, 1)

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrAlreadyMigrated)
}
