package keyring

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestParseProcKeysTimeout(t *testing.T) {
	cases := map[string]time.Duration{"45s": 45 * time.Second, "9m": 9 * time.Minute, "2h": 2 * time.Hour, "1d": 24 * time.Hour, "1w": 7 * 24 * time.Hour}
	for in, want := range cases {
		if got, ok := parseProcKeysTimeout(in); !ok || got != want {
			t.Errorf("%s: got %v ok=%v", in, got, ok)
		}
	}
	for _, in := range []string{"perm", "expd", "", "s", "0s", "xs"} {
		if _, ok := parseProcKeysTimeout(in); ok {
			t.Errorf("%q should not parse", in)
		}
	}
}

func kernelKeyringOrSkip(t *testing.T, key string) kernelBackend {
	t.Helper()
	b := kernelBackend{}
	if err := b.set(key, "probe", time.Minute); err != nil {
		t.Skipf("kernel keyring unavailable: %v", err)
	}
	return b
}

func TestKernelBackendRoundTrip(t *testing.T) {
	key := fmt.Sprintf("%stest-%d-roundtrip", keyPrefix, os.Getpid())
	b := kernelKeyringOrSkip(t, key)
	defer b.forget(key)

	if err := b.set(key, "s3cret", time.Minute); err != nil {
		t.Fatal(err)
	}
	if got, err := b.get(key); err != nil || got != "s3cret" {
		t.Fatalf("get: %q %v", got, err)
	}
	if d, ok := b.expiresIn(key); !ok || d <= 0 || d > time.Minute {
		t.Errorf("expiresIn: %v %v", d, ok)
	}
	if err := b.forget(key); err != nil {
		t.Fatal(err)
	}
	if _, err := b.get(key); err == nil {
		t.Error("forgotten key should not be readable")
	}
	if err := b.forget(key); err != nil {
		t.Errorf("forgetting a missing key should succeed: %v", err)
	}
}

func TestKernelBackendExpires(t *testing.T) {
	key := fmt.Sprintf("%stest-%d-expiry", keyPrefix, os.Getpid())
	b := kernelKeyringOrSkip(t, key)
	defer b.forget(key)

	if err := b.set(key, "s3cret", time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	if _, err := b.get(key); err == nil {
		t.Error("key should expire without any reader deleting it")
	}
}

func TestKernelBackendForgetAll(t *testing.T) {
	first := fmt.Sprintf("%stest-%d-all-1", keyPrefix, os.Getpid())
	second := fmt.Sprintf("%stest-%d-all-2", keyPrefix, os.Getpid())
	b := kernelKeyringOrSkip(t, first)
	defer b.forget(first)
	defer b.forget(second)

	if err := b.set(second, "two", time.Minute); err != nil {
		t.Fatal(err)
	}
	n, err := forgetWithPrefix(fmt.Sprintf("%stest-%d-all-", keyPrefix, os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("removed %d keys, want 2", n)
	}
	for _, key := range []string{first, second} {
		if _, err := b.get(key); err == nil {
			t.Errorf("%s should be gone", key)
		}
	}
}
