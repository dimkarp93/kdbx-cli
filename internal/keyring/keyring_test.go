package keyring

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func TestNewCache(t *testing.T) {
	if New(nil).enabled {
		t.Error("nil cache config should be disabled")
	}
	if New(&config.CacheConfig{Enabled: false}).enabled {
		t.Error("enabled=false should be disabled")
	}
	c := New(&config.CacheConfig{Enabled: true})
	if !c.enabled || c.ttl != defaultCacheTTL {
		t.Errorf("default ttl: %+v", c)
	}
	c = New(&config.CacheConfig{Enabled: true, TTL: "30s"})
	if c.ttl != 30*time.Second {
		t.Errorf("parsed ttl: %v", c.ttl)
	}
	c = New(&config.CacheConfig{Enabled: true, TTL: "garbage"})
	if c.ttl != defaultCacheTTL {
		t.Errorf("invalid ttl should fall back to default: %v", c.ttl)
	}
}

type fakeBackend struct {
	keys map[string]string
	ttls map[string]time.Duration
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{keys: map[string]string{}, ttls: map[string]time.Duration{}}
}

func (f *fakeBackend) get(key string) (string, error) {
	v, ok := f.keys[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}

func (f *fakeBackend) set(key, password string, ttl time.Duration) error {
	f.keys[key] = password
	f.ttls[key] = ttl
	return nil
}

func (f *fakeBackend) forget(key string) error {
	delete(f.keys, key)
	return nil
}

func (f *fakeBackend) forgetAll() (int, error) {
	n := len(f.keys)
	f.keys = map[string]string{}
	return n, nil
}

func (f *fakeBackend) expiresIn(key string) (time.Duration, bool) {
	if _, ok := f.keys[key]; !ok {
		return 0, false
	}
	return f.ttls[key], true
}

func withFakeBackend(t *testing.T) *fakeBackend {
	t.Helper()
	fake := newFakeBackend()
	origStore, origDelete := store, keyringDelete
	store = fake
	keyringDelete = func(_, _ string) error { return nil }
	t.Cleanup(func() { store, keyringDelete = origStore, origDelete })
	return fake
}

func TestCacheHitStatusForget(t *testing.T) {
	fake := withFakeBackend(t)
	c := Cache{enabled: true, ttl: time.Minute}

	if _, ok := c.Get("/k.kdbx"); ok {
		t.Error("empty cache should miss")
	}
	c.Remember("/k.kdbx", "pw")
	if pw, ok := c.Get("/k.kdbx"); !ok || pw != "pw" {
		t.Errorf("hit: ok=%v pw=%q", ok, pw)
	}
	if got := fake.ttls[storeKey("/k.kdbx")]; got != time.Minute {
		t.Errorf("ttl passed to backend: %v", got)
	}
	if st := c.Status("/k.kdbx"); !st.Cached || st.ExpiresIn != time.Minute {
		t.Errorf("status: %+v", st)
	}

	disabled := Cache{}
	disabled.Remember("/x", "y")
	if _, ok := disabled.Get("/x"); ok {
		t.Error("disabled cache should never hit")
	}

	c.Forget("/k.kdbx")
	if _, ok := c.Get("/k.kdbx"); ok {
		t.Error("forget should remove the entry")
	}
	if st := c.Status("/k.kdbx"); st.Cached {
		t.Error("forgotten entry should not be reported as cached")
	}
}

func TestStoreKeyIsCanonical(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.kdbx")
	if err := os.WriteFile(real, nil, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.kdbx")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	want := keyPrefix + resolved
	if got := storeKey(link); got != want {
		t.Errorf("symlink: got %q, want %q", got, want)
	}
	t.Chdir(dir)
	if got := storeKey("real.kdbx"); got != want {
		t.Errorf("relative: got %q, want %q", got, want)
	}
	if got := storeKey("/no/such/file.kdbx"); got != keyPrefix+"/no/such/file.kdbx" {
		t.Errorf("missing file: got %q", got)
	}
}

func TestForgetAll(t *testing.T) {
	fake := withFakeBackend(t)
	legacyCleared := false
	origDeleteAll := keyringDeleteAll
	keyringDeleteAll = func(string) error { legacyCleared = true; return nil }
	defer func() { keyringDeleteAll = origDeleteAll }()

	c := Cache{enabled: true, ttl: time.Minute}
	c.Remember("/a.kdbx", "a")
	c.Remember("/b.kdbx", "b")
	n, err := ForgetAll()
	if err != nil || n != 2 {
		t.Errorf("ForgetAll: n=%d err=%v", n, err)
	}
	if len(fake.keys) != 0 {
		t.Errorf("keys left: %v", fake.keys)
	}
	if _, ok := c.Get("/a.kdbx"); ok {
		t.Error("cache should be empty")
	}
	if runtime.GOOS == "linux" && !legacyCleared {
		t.Error("legacy Secret Service entries should be cleared on Linux")
	}
}

func TestMergeTitles(t *testing.T) {
	got := mergeTitles([]string{"b", "a"}, []string{"c", "a"})
	if !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("merge: %v", got)
	}
	if got := mergeTitles(nil, nil); len(got) != 0 {
		t.Errorf("empty merge: %v", got)
	}
}

func TestDecodeEntry(t *testing.T) {
	cases := map[string]entry{
		"plain-pw":                             {Password: "plain-pw"},
		`{"password":"pw"}`:                    {Password: "pw"},
		`{"password":"pw","titles":["a","b"]}`: {Password: "pw", Titles: []string{"a", "b"}},
		`{"titles":["a"]}`:                     {Password: `{"titles":["a"]}`},
		`{broken`:                              {Password: `{broken`},
	}
	for raw, want := range cases {
		if got := decodeEntry(raw); !reflect.DeepEqual(got, want) {
			t.Errorf("decode %q: got %+v, want %+v", raw, got, want)
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	want := entry{Password: "p\"w{", Titles: []string{"a", "b"}}
	if got := decodeEntry(encodeEntry(want)); !reflect.DeepEqual(got, want) {
		t.Errorf("round trip: %+v", got)
	}
}

func TestGrantMergesAndRememberKeepsTitles(t *testing.T) {
	withFakeBackend(t)
	c := Cache{enabled: true, ttl: time.Minute}

	c.Grant("/k.kdbx", "pw", []string{"b"})
	c.Grant("/k.kdbx", "pw", []string{"a", "b"})
	if got := c.Granted("/k.kdbx"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("granted: %v", got)
	}

	c.Remember("/k.kdbx", "new-pw")
	if pw, ok := c.Get("/k.kdbx"); !ok || pw != "new-pw" {
		t.Errorf("remember: ok=%v pw=%q", ok, pw)
	}
	if got := c.Granted("/k.kdbx"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("remember dropped titles: %v", got)
	}

	c.Forget("/k.kdbx")
	if got := c.Granted("/k.kdbx"); len(got) != 0 {
		t.Errorf("forget should drop titles: %v", got)
	}
}

func TestGrantDisabledCache(t *testing.T) {
	fake := withFakeBackend(t)
	Cache{}.Grant("/k.kdbx", "pw", []string{"a"})
	if len(fake.keys) != 0 {
		t.Error("disabled cache must not store anything")
	}
	if got := (Cache{}).Granted("/k.kdbx"); got != nil {
		t.Errorf("disabled cache granted: %v", got)
	}
}

func TestLegacyRawEntryReadable(t *testing.T) {
	fake := withFakeBackend(t)
	c := Cache{enabled: true, ttl: time.Minute}
	fake.keys[storeKey("/k.kdbx")] = "legacy-pw"
	if pw, ok := c.Get("/k.kdbx"); !ok || pw != "legacy-pw" {
		t.Errorf("legacy: ok=%v pw=%q", ok, pw)
	}
	if got := c.Granted("/k.kdbx"); len(got) != 0 {
		t.Errorf("legacy granted: %v", got)
	}
}
