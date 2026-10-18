package keyring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

const (
	keyringService  = "kdbx-cli"
	defaultCacheTTL = 10 * time.Minute
)

type systemKeyring struct{}

func (systemKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (systemKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemKeyring) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

func (systemKeyring) DeleteAll(service string) error {
	return keyring.DeleteAll(service)
}

type backend interface {
	get(key string) (string, error)
	set(key, password string, ttl time.Duration) error
	forget(key string) error
	forgetAll() (int, error)
	expiresIn(key string) (time.Duration, bool)
}

type storage struct {
	backend backend
	legacy  keyring.Keyring
}

func systemStorage() storage {
	return storage{backend: newBackend(systemKeyring{}), legacy: systemKeyring{}}
}

func (s storage) forgetAll() (int, error) {
	n, err := s.backend.forgetAll()
	dropAllLegacy(s.legacy)
	return n, err
}

const keyPrefix = keyringService + ":"

func storeKey(keyStore string) string {
	path := config.ExpandHome(keyStore)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	return keyPrefix + path
}

type Cache struct {
	enabled bool
	ttl     time.Duration
	storage storage
}

func New(c *config.CacheConfig) Cache {
	return newCache(c, systemStorage())
}

func newCache(c *config.CacheConfig, s storage) Cache {
	if c == nil || !c.Enabled {
		return Cache{storage: s}
	}
	ttl := defaultCacheTTL
	if c.TTL != "" {
		if d, err := time.ParseDuration(c.TTL); err == nil && d > 0 {
			ttl = d
		}
	}
	return Cache{enabled: true, ttl: ttl, storage: s}
}

func (c Cache) Enabled() bool {
	return c.enabled
}

func (c Cache) Get(keyStore string) (string, bool) {
	if !c.enabled {
		return "", false
	}
	raw, err := c.storage.backend.get(storeKey(keyStore))
	if err != nil {
		return "", false
	}
	pw := decodeEntry(raw).Password
	if pw == "" {
		return "", false
	}
	return pw, true
}

func (c Cache) Remember(keyStore, password string) {
	if !c.enabled {
		return
	}
	c.Grant(keyStore, password, nil)
}

func (c Cache) Forget(keyStore string) {
	_ = c.storage.backend.forget(storeKey(keyStore))
	dropLegacy(c.storage.legacy, keyStore)
}

type CacheStatus struct {
	Cached    bool
	ExpiresIn time.Duration
}

func (c Cache) Status(keyStore string) CacheStatus {
	if !c.enabled {
		return CacheStatus{}
	}
	remaining, ok := c.storage.backend.expiresIn(storeKey(keyStore))
	if !ok {
		return CacheStatus{}
	}
	return CacheStatus{Cached: true, ExpiresIn: remaining}
}

func ForgetAll() (int, error) {
	return systemStorage().forgetAll()
}

type entry struct {
	Password string   `json:"password"`
	Titles   []string `json:"titles,omitempty"`
}

func encodeEntry(e entry) string {
	data, _ := json.Marshal(e)
	return string(data)
}

func decodeEntry(raw string) entry {
	var e entry
	if err := json.Unmarshal([]byte(raw), &e); err != nil || e.Password == "" {
		return entry{Password: raw}
	}
	return e
}

func mergeTitles(current, added []string) []string {
	seen := make(map[string]bool, len(current)+len(added))
	var out []string
	for _, title := range append(append([]string{}, current...), added...) {
		if !seen[title] {
			seen[title] = true
			out = append(out, title)
		}
	}
	sort.Strings(out)
	return out
}

func (c Cache) Granted(keyStore string) []string {
	if !c.enabled {
		return nil
	}
	raw, err := c.storage.backend.get(storeKey(keyStore))
	if err != nil {
		return nil
	}
	return decodeEntry(raw).Titles
}

func (c Cache) Grant(keyStore, password string, titles []string) {
	if !c.enabled {
		return
	}
	merged := mergeTitles(c.Granted(keyStore), titles)
	if err := c.storage.backend.set(storeKey(keyStore), encodeEntry(entry{Password: password, Titles: merged}), c.ttl); err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot cache password:", err)
	}
}
