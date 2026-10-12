package keyring

import "time"

const (
	kernelKeyType = "user"
	kernelKeyPerm = 0x3f010000
)

type kernelBackend struct{}

var procKeysPath = "/proc/keys"

var procKeysUnits = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour,
}

func newBackend() backend {
	panic("not implemented")
}

func dropLegacy(keyStore string) {
	panic("not implemented")
}

func (kernelBackend) get(key string) (string, error) {
	panic("not implemented")
}

func (kernelBackend) set(key, password string, ttl time.Duration) error {
	panic("not implemented")
}

func (kernelBackend) forget(key string) error {
	panic("not implemented")
}

func (kernelBackend) forgetAll() (int, error) {
	panic("not implemented")
}

func (kernelBackend) expiresIn(key string) (time.Duration, bool) {
	panic("not implemented")
}
