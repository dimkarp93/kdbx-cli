//go:build !linux

package keyring

import "time"

type secretServiceBackend struct{}

func newBackend() backend {
	panic("not implemented")
}

func dropLegacy(keyStore string) {}

type expiringSecret struct {
	Password  string `json:"password"`
	ExpiresAt int64  `json:"expires_at"`
}

func (secretServiceBackend) get(key string) (string, error) {
	panic("not implemented")
}

func (secretServiceBackend) set(key, password string, ttl time.Duration) error {
	panic("not implemented")
}

func (secretServiceBackend) forget(key string) error {
	panic("not implemented")
}

func (secretServiceBackend) forgetAll() (int, error) {
	panic("not implemented")
}

func (secretServiceBackend) expiresIn(key string) (time.Duration, bool) {
	panic("not implemented")
}
