package keyring

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	kernelKeyType = "user"
	kernelKeyPerm = 0x3f010000
)

type kernelBackend struct{}

func newBackend() backend {
	panic("not implemented")
}

func dropLegacy(keyStore string) {
	panic("not implemented")
}

func searchKey(key string) (int, error) {
	return unix.KeyctlSearch(unix.KEY_SPEC_USER_KEYRING, kernelKeyType, key, 0)
}

func invalidateKey(id int) error {
	_, err := unix.KeyctlInt(unix.KEYCTL_INVALIDATE, id, 0, 0, 0)
	return err
}

func (kernelBackend) get(key string) (string, error) {
	id, err := searchKey(key)
	if err != nil {
		return "", err
	}
	buf := make([]byte, 256)
	for {
		n, err := unix.KeyctlBuffer(unix.KEYCTL_READ, id, buf, 0)
		if err != nil {
			return "", err
		}
		if n <= len(buf) {
			return string(buf[:n]), nil
		}
		buf = make([]byte, n)
	}
}

func (kernelBackend) set(key, password string, ttl time.Duration) error {
	id, err := unix.AddKey(kernelKeyType, key, []byte(password), unix.KEY_SPEC_USER_KEYRING)
	if err != nil {
		return err
	}
	seconds := max(int(ttl.Round(time.Second)/time.Second), 1)
	if _, err := unix.KeyctlInt(unix.KEYCTL_SET_TIMEOUT, id, seconds, 0, 0); err != nil {
		_ = invalidateKey(id)
		return err
	}
	return unix.KeyctlSetperm(id, kernelKeyPerm)
}

func (kernelBackend) forget(key string) error {
	id, err := searchKey(key)
	if errors.Is(err, unix.ENOKEY) || errors.Is(err, unix.EKEYEXPIRED) {
		return nil
	}
	if err != nil {
		return err
	}
	return invalidateKey(id)
}

func userKeyringIDs() ([]int, error) {
	buf := make([]byte, 1024)
	for {
		n, err := unix.KeyctlBuffer(unix.KEYCTL_READ, unix.KEY_SPEC_USER_KEYRING, buf, 0)
		if err != nil {
			return nil, err
		}
		if n <= len(buf) {
			ids := make([]int, 0, n/4)
			for i := 0; i+4 <= n; i += 4 {
				ids = append(ids, int(int32(binary.NativeEndian.Uint32(buf[i:i+4]))))
			}
			return ids, nil
		}
		buf = make([]byte, n)
	}
}

func (kernelBackend) forgetAll() (int, error) {
	ids, err := userKeyringIDs()
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, id := range ids {
		desc, err := unix.KeyctlString(unix.KEYCTL_DESCRIBE, id)
		if err != nil {
			continue
		}
		parts := strings.SplitN(desc, ";", 5)
		if len(parts) != 5 || parts[0] != kernelKeyType || !strings.HasPrefix(parts[4], keyPrefix) {
			continue
		}
		if invalidateKey(id) == nil {
			removed++
		}
	}
	return removed, nil
}

var procKeysPath = "/proc/keys"

var procKeysUnits = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour,
}

func parseProcKeysTimeout(field string) (time.Duration, bool) {
	if len(field) < 2 {
		return 0, false
	}
	unit, ok := procKeysUnits[field[len(field)-1]]
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(field[:len(field)-1])
	if err != nil || n <= 0 {
		return 0, false
	}
	return time.Duration(n) * unit, true
}

func (kernelBackend) expiresIn(key string) (time.Duration, bool) {
	id, err := searchKey(key)
	if err != nil {
		return 0, false
	}
	data, err := os.ReadFile(procKeysPath)
	if err != nil {
		return 0, false
	}
	prefix := fmt.Sprintf("%08x ", id)
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			return 0, false
		}
		return parseProcKeysTimeout(fields[3])
	}
	return 0, false
}
