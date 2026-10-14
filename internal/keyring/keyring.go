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

var (
	keyringSet       = keyring.Set
	keyringGet       = keyring.Get
	keyringDelete    = keyring.Delete
	keyringDeleteAll = keyring.DeleteAll
)

type backend interface {
	get(key string) (string, error)
	set(key, password string, ttl time.Duration) error
	forget(key string) error
	forgetAll() (int, error)
	expiresIn(key string) (time.Duration, bool)
}

var store backend = newBackend()

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
}

func New(c *config.CacheConfig) Cache {
	if c == nil || !c.Enabled {
		return Cache{}
	}
	ttl := defaultCacheTTL
	if c.TTL != "" {
		if d, err := time.ParseDuration(c.TTL); err == nil && d > 0 {
			ttl = d
		}
	}
	return Cache{enabled: true, ttl: ttl}
}

func (c Cache) Enabled() bool {
	return c.enabled
}

func (c Cache) Get(keyStore string) (string, bool) {
	if !c.enabled {
		return "", false
	}
	raw, err := store.get(storeKey(keyStore))
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
	_ = store.forget(storeKey(keyStore))
	dropLegacy(keyStore)
}

type CacheStatus struct {
	Cached    bool
	ExpiresIn time.Duration
}

func (c Cache) Status(keyStore string) CacheStatus {
	if !c.enabled {
		return CacheStatus{}
	}
	remaining, ok := store.expiresIn(storeKey(keyStore))
	if !ok {
		return CacheStatus{}
	}
	return CacheStatus{Cached: true, ExpiresIn: remaining}
}

func ForgetAll() (int, error) {
	n, err := store.forgetAll()
	dropAllLegacy()
	return n, err
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
	raw, err := store.get(storeKey(keyStore))
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
	if err := store.set(storeKey(keyStore), encodeEntry(entry{Password: password, Titles: merged}), c.ttl); err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot cache password:", err)
	}
}
