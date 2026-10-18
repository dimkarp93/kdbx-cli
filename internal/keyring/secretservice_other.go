//go:build !linux

package keyring

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/zalando/go-keyring"
)

type secretServiceBackend struct {
	service keyring.Keyring
}

func newBackend(service keyring.Keyring) backend { return secretServiceBackend{service: service} }

func dropLegacy(keyring.Keyring, string) {}

type expiringSecret struct {
	Password  string `json:"password"`
	ExpiresAt int64  `json:"expires_at"`
}

func (b secretServiceBackend) read(key string) (expiringSecret, error) {
	raw, err := b.service.Get(keyringService, key)
	if err != nil {
		return expiringSecret{}, err
	}
	var es expiringSecret
	if err := json.Unmarshal([]byte(raw), &es); err != nil || es.ExpiresAt == 0 {
		return expiringSecret{}, errors.New("malformed cache entry")
	}
	return es, nil
}

func (b secretServiceBackend) get(key string) (string, error) {
	es, err := b.read(key)
	if err != nil {
		return "", err
	}
	if time.Now().Unix() >= es.ExpiresAt {
		_ = b.service.Delete(keyringService, key)
		return "", errors.New("cache entry expired")
	}
	return es.Password, nil
}

func (b secretServiceBackend) set(key, password string, ttl time.Duration) error {
	data, _ := json.Marshal(expiringSecret{Password: password, ExpiresAt: time.Now().Add(ttl).Unix()})
	return b.service.Set(keyringService, key, string(data))
}

func (b secretServiceBackend) forget(key string) error {
	return b.service.Delete(keyringService, key)
}

func (b secretServiceBackend) forgetAll() (int, error) {
	return -1, b.service.DeleteAll(keyringService)
}

func (b secretServiceBackend) expiresIn(key string) (time.Duration, bool) {
	es, err := b.read(key)
	if err != nil {
		return 0, false
	}
	remaining := time.Until(time.Unix(es.ExpiresAt, 0))
	if remaining <= 0 {
		_ = b.service.Delete(keyringService, key)
		return 0, false
	}
	return remaining, true
}

func dropAllLegacy(keyring.Keyring) {}
