package keyring

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

type fakeBackend struct {
	keys map[string]string
	ttls map[string]time.Duration
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{keys: map[string]string{}, ttls: map[string]time.Duration{}}
}

func (f *fakeBackend) get(key string) (string, error) {
	v, ok := f.keys[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (f *fakeBackend) set(key, password string, ttl time.Duration) error {
	f.keys[key] = password
	f.ttls[key] = ttl
	return nil
}

func (f *fakeBackend) forget(key string) error {
	delete(f.keys, key)
	return nil
}

func (f *fakeBackend) forgetAll() (int, error) {
	n := len(f.keys)
	f.keys = map[string]string{}
	return n, nil
}

func (f *fakeBackend) expiresIn(key string) (time.Duration, bool) {
	if _, ok := f.keys[key]; !ok {
		return 0, false
	}
	return f.ttls[key], true
}

type fakeLegacyKeyring struct {
	deletedAll bool
}

func (*fakeLegacyKeyring) Set(_, _, _ string) error {
	return nil
}

func (*fakeLegacyKeyring) Get(_, _ string) (string, error) {
	return "", errors.New("not found")
}

func (*fakeLegacyKeyring) Delete(_, _ string) error {
	return nil
}

func (f *fakeLegacyKeyring) DeleteAll(string) error {
	f.deletedAll = true
	return nil
}

type fakeStorage struct {
	backend *fakeBackend
	legacy  *fakeLegacyKeyring
}

func newFakeStorage() fakeStorage {
	return fakeStorage{backend: newFakeBackend(), legacy: &fakeLegacyKeyring{}}
}

func (f fakeStorage) storage() storage {
	return storage{backend: f.backend, legacy: f.legacy}
}

func (f fakeStorage) enabledCache() Cache {
	return Cache{enabled: true, ttl: time.Minute, storage: f.storage()}
}

func (f fakeStorage) disabledCache() Cache {
	return Cache{storage: f.storage()}
}

func TestNewCache(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.CacheConfig
		want Cache
	}{
		{"nil config", nil, Cache{}},
		{"disabled", &config.CacheConfig{Enabled: false, TTL: "30s"}, Cache{}},
		{"default ttl", &config.CacheConfig{Enabled: true}, Cache{enabled: true, ttl: defaultCacheTTL}},
		{"parsed ttl", &config.CacheConfig{Enabled: true, TTL: "30s"}, Cache{enabled: true, ttl: 30 * time.Second}},
		{"invalid ttl", &config.CacheConfig{Enabled: true, TTL: "garbage"}, Cache{enabled: true, ttl: defaultCacheTTL}},
		{"negative ttl", &config.CacheConfig{Enabled: true, TTL: "-1m"}, Cache{enabled: true, ttl: defaultCacheTTL}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, newCache(c.cfg, storage{}))
		})
	}
}

func TestCacheMissWhenEmpty(t *testing.T) {
	_, ok := newFakeStorage().enabledCache().Get("/k.kdbx")
	assert.False(t, ok)
}

func TestCacheRememberThenGet(t *testing.T) {
	fake := newFakeStorage()
	c := fake.enabledCache()
	c.Remember("/k.kdbx", "pw")

	pw, ok := c.Get("/k.kdbx")

	t.Run("hits", func(t *testing.T) {
		require.True(t, ok)
		assert.Equal(t, "pw", pw)
	})
	t.Run("passes ttl to backend", func(t *testing.T) {
		assert.Equal(t, time.Minute, fake.backend.ttls[storeKey("/k.kdbx")])
	})
	t.Run("status", func(t *testing.T) {
		assert.Equal(t, CacheStatus{Cached: true, ExpiresIn: time.Minute}, c.Status("/k.kdbx"))
	})
}

func TestCacheDisabledNeverHits(t *testing.T) {
	disabled := newFakeStorage().disabledCache()
	disabled.Remember("/x", "y")

	_, ok := disabled.Get("/x")
	assert.False(t, ok)
}

func TestCacheForget(t *testing.T) {
	c := newFakeStorage().enabledCache()
	c.Remember("/k.kdbx", "pw")

	c.Forget("/k.kdbx")

	t.Run("misses", func(t *testing.T) {
		_, ok := c.Get("/k.kdbx")
		assert.False(t, ok)
	})
	t.Run("not reported as cached", func(t *testing.T) {
		assert.False(t, c.Status("/k.kdbx").Cached)
	})
}

func canonicalStore(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	real := filepath.Join(dir, "real.kdbx")
	require.NoError(t, os.WriteFile(real, nil, 0600))

	resolved, err := filepath.EvalSymlinks(real)
	require.NoError(t, err)

	return dir, keyPrefix + resolved
}

func TestStoreKeyResolvesSymlink(t *testing.T) {
	dir, want := canonicalStore(t)
	link := filepath.Join(dir, "link.kdbx")
	require.NoError(t, os.Symlink(filepath.Join(dir, "real.kdbx"), link))

	assert.Equal(t, want, storeKey(link))
}

func TestStoreKeyResolvesRelativePath(t *testing.T) {
	dir, want := canonicalStore(t)
	t.Chdir(dir)

	assert.Equal(t, want, storeKey("real.kdbx"))
}

func TestStoreKeyKeepsMissingPath(t *testing.T) {
	assert.Equal(t, keyPrefix+"/no/such/file.kdbx", storeKey("/no/such/file.kdbx"))
}

func TestStorageForgetAll(t *testing.T) {
	fake := newFakeStorage()
	c := fake.enabledCache()
	c.Remember("/a.kdbx", "a")
	c.Remember("/b.kdbx", "b")

	n, err := fake.storage().forgetAll()

	t.Run("reports removed count", func(t *testing.T) {
		require.NoError(t, err)
		assert.Equal(t, 2, n)
	})
	t.Run("backend empty", func(t *testing.T) {
		assert.Empty(t, fake.backend.keys)
	})
	t.Run("legacy entries cleared on linux", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("legacy Secret Service cleanup is linux-only")
		}
		assert.True(t, fake.legacy.deletedAll)
	})
}

func TestMergeTitles(t *testing.T) {
	cases := []struct {
		name           string
		current, added []string
		want           []string
	}{
		{"sorted union", []string{"b", "a"}, []string{"c", "a"}, []string{"a", "b", "c"}},
		{"empty", nil, nil, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, mergeTitles(c.current, c.added))
		})
	}
}

func TestDecodeEntry(t *testing.T) {
	cases := map[string]entry{
		"plain-pw":                             {Password: "plain-pw"},
		`{"password":"pw"}`:                    {Password: "pw"},
		`{"password":"pw","titles":["a","b"]}`: {Password: "pw", Titles: []string{"a", "b"}},
		`{"titles":["a"]}`:                     {Password: `{"titles":["a"]}`},
		`{broken`:                              {Password: `{broken`},
	}

	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			assert.Equal(t, want, decodeEntry(raw))
		})
	}
}

func TestGrantMergesTitles(t *testing.T) {
	c := newFakeStorage().enabledCache()

	c.Grant("/k.kdbx", "pw", []string{"b"})
	c.Grant("/k.kdbx", "pw", []string{"a", "b"})

	assert.Equal(t, []string{"a", "b"}, c.Granted("/k.kdbx"))
}

func TestRememberKeepsGrantedTitles(t *testing.T) {
	c := newFakeStorage().enabledCache()
	c.Grant("/k.kdbx", "pw", []string{"a", "b"})

	c.Remember("/k.kdbx", "new-pw")

	t.Run("updates password", func(t *testing.T) {
		pw, ok := c.Get("/k.kdbx")
		require.True(t, ok)
		assert.Equal(t, "new-pw", pw)
	})
	t.Run("keeps titles", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, c.Granted("/k.kdbx"))
	})
}

func TestForgetDropsGrantedTitles(t *testing.T) {
	c := newFakeStorage().enabledCache()
	c.Grant("/k.kdbx", "pw", []string{"a"})

	c.Forget("/k.kdbx")

	assert.Empty(t, c.Granted("/k.kdbx"))
}

func TestGrantDisabledCache(t *testing.T) {
	fake := newFakeStorage()
	disabled := fake.disabledCache()

	disabled.Grant("/k.kdbx", "pw", []string{"a"})

	t.Run("stores nothing", func(t *testing.T) {
		assert.Empty(t, fake.backend.keys)
	})
	t.Run("grants nothing", func(t *testing.T) {
		assert.Nil(t, disabled.Granted("/k.kdbx"))
	})
}

func TestLegacyRawEntryReadable(t *testing.T) {
	fake := newFakeStorage()
	fake.backend.keys[storeKey("/k.kdbx")] = "legacy-pw"
	c := fake.enabledCache()

	t.Run("password", func(t *testing.T) {
		pw, ok := c.Get("/k.kdbx")
		require.True(t, ok)
		assert.Equal(t, "legacy-pw", pw)
	})
	t.Run("no granted titles", func(t *testing.T) {
		assert.Empty(t, c.Granted("/k.kdbx"))
	})
}
