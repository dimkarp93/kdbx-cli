package keyring

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProcKeysTimeout(t *testing.T) {
	cases := map[string]time.Duration{"45s": 45 * time.Second, "9m": 9 * time.Minute, "2h": 2 * time.Hour, "1d": 24 * time.Hour, "1w": 7 * 24 * time.Hour}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got, ok := parseProcKeysTimeout(in)
			require.True(t, ok)
			assert.Equal(t, want, got)
		})
	}
}

func TestParseProcKeysTimeoutRejects(t *testing.T) {
	for _, in := range []string{"perm", "expd", "", "s", "0s", "xs"} {
		t.Run(in, func(t *testing.T) {
			_, ok := parseProcKeysTimeout(in)
			assert.False(t, ok)
		})
	}
}

func testKey(suffix string) string {
	return fmt.Sprintf("%stest-%d-%s", keyPrefix, os.Getpid(), suffix)
}

func kernelKeyringOrSkip(t *testing.T, key string) kernelBackend {
	t.Helper()

	b := kernelBackend{}
	if err := b.set(key, "probe", time.Minute); err != nil {
		t.Skipf("kernel keyring unavailable: %v", err)
	}

	t.Cleanup(func() { _ = b.forget(key) })
	return b
}

func TestKernelBackendRoundTrip(t *testing.T) {
	key := testKey("roundtrip")
	b := kernelKeyringOrSkip(t, key)

	setErr := b.set(key, "s3cret", time.Minute)
	got, getErr := b.get(key)
	expiresIn, hasTimeout := b.expiresIn(key)

	t.Run("stores", func(t *testing.T) {
		assert.NoError(t, setErr)
	})
	t.Run("reads back", func(t *testing.T) {
		require.NoError(t, getErr)
		assert.Equal(t, "s3cret", got)
	})
	t.Run("reports timeout within ttl", func(t *testing.T) {
		require.True(t, hasTimeout)
		assert.Greater(t, expiresIn, time.Duration(0))
		assert.LessOrEqual(t, expiresIn, time.Minute)
	})
}

func TestKernelBackendForget(t *testing.T) {
	key := testKey("forget")
	b := kernelKeyringOrSkip(t, key)

	forgetErr := b.forget(key)
	_, getErr := b.get(key)

	t.Run("succeeds", func(t *testing.T) {
		assert.NoError(t, forgetErr)
	})
	t.Run("key unreadable", func(t *testing.T) {
		assert.Error(t, getErr)
	})
	t.Run("forgetting missing key succeeds", func(t *testing.T) {
		assert.NoError(t, b.forget(key))
	})
}

func TestKernelBackendExpires(t *testing.T) {
	key := testKey("expiry")
	b := kernelKeyringOrSkip(t, key)
	require.NoError(t, b.set(key, "s3cret", time.Second))

	time.Sleep(2 * time.Second)
	_, err := b.get(key)

	assert.Error(t, err)
}

func TestKernelBackendForgetAll(t *testing.T) {
	first, second := testKey("all-1"), testKey("all-2")
	b := kernelKeyringOrSkip(t, first)
	setErr := b.set(second, "two", time.Minute)
	t.Cleanup(func() { _ = b.forget(second) })

	n, err := forgetWithPrefix(testKey("all-"))

	t.Run("stores second key", func(t *testing.T) {
		require.NoError(t, setErr)
	})
	t.Run("removes both keys", func(t *testing.T) {
		require.NoError(t, err)
		assert.Equal(t, 2, n)
	})
	for _, key := range []string{first, second} {
		t.Run(key+" gone", func(t *testing.T) {
			_, getErr := b.get(key)
			assert.Error(t, getErr)
		})
	}
}
