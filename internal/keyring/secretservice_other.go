//go:build !linux

package keyring

import (
	"encoding/json"
	"errors"
	"time"
)

type secretServiceBackend struct{}

func newBackend() backend { return secretServiceBackend{} }

func dropLegacy(keyStore string) {}

type expiringSecret struct {
	Password  string `json:"password"`
	ExpiresAt int64  `json:"expires_at"`
}

func (secretServiceBackend) read(key string) (expiringSecret, error) {
	raw, err := keyringGet(keyringService, key)
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
		_ = keyringDelete(keyringService, key)
		return "", errors.New("cache entry expired")
	}
	return es.Password, nil
}

func (secretServiceBackend) set(key, password string, ttl time.Duration) error {
	data, _ := json.Marshal(expiringSecret{Password: password, ExpiresAt: time.Now().Add(ttl).Unix()})
	return keyringSet(keyringService, key, string(data))
}

func (secretServiceBackend) forget(key string) error {
	return keyringDelete(keyringService, key)
}

func (secretServiceBackend) forgetAll() (int, error) {
	return -1, keyringDeleteAll(keyringService)
}

func (b secretServiceBackend) expiresIn(key string) (time.Duration, bool) {
	es, err := b.read(key)
	if err != nil {
		return 0, false
	}
	remaining := time.Until(time.Unix(es.ExpiresAt, 0))
	if remaining <= 0 {
		_ = keyringDelete(keyringService, key)
		return 0, false
	}
	return remaining, true
}
