package keyring

import (
	"fmt"
	"os"
	"path/filepath"
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

type cachedSecret struct {
	Password string `json:"password"`
	StoredAt int64  `json:"stored_at"`
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

func (c Cache) Get(keyStore string) (string, bool) {
	if !c.enabled {
		return "", false
	}
	pw, err := store.get(storeKey(keyStore))
	if err != nil || pw == "" {
		return "", false
	}
	return pw, true
}

func (c Cache) Remember(keyStore, password string) {
	if !c.enabled {
		return
	}
	if err := store.set(storeKey(keyStore), password, c.ttl); err != nil {
		fmt.Fprintln(os.Stderr, "warning: cannot cache password:", err)
	}
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
