package keepass

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<KeePassFile><Root><Group><Name>Root</Name>
  <Entry><String><Key>Title</Key><Value>GITHUB_TOKEN</Value></String><String><Key>Password</Key><Value>ghp_top</Value></String></Entry>
  <Group><Name>web</Name>
    <Entry><String><Key>Title</Key><Value>API_KEY</Value></String><String><Key>Password</Key><Value>web_secret</Value></String></Entry>
    <Entry><String><Key>Title</Key><Value>DUP</Value></String><String><Key>Password</Key><Value>web_dup</Value></String></Entry>
  </Group>
  <Group><Name>db</Name>
    <Entry><String><Key>Title</Key><Value>DUP</Value></String><String><Key>Password</Key><Value>db_dup</Value></String></Entry>
  </Group>
</Group></Root></KeePassFile>`

func parseSample(t *testing.T) []Entry {
	t.Helper()
	entries, err := ParseSecrets(sampleXML)
	require.NoError(t, err)
	return entries
}

func TestLookupSecretFound(t *testing.T) {
	entries := parseSample(t)
	cases := map[string]string{
		"GITHUB_TOKEN": "ghp_top",
		"API_KEY":      "web_secret",
		"web/API_KEY":  "web_secret",
		"db/DUP":       "db_dup",
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := LookupSecret(entries, name)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestLookupSecretFails(t *testing.T) {
	entries := parseSample(t)
	cases := map[string]string{
		"ambiguous title": "DUP",
		"missing title":   "MISSING",
	}

	for name, title := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := LookupSecret(entries, title)
			assert.Error(t, err)
		})
	}
}

func engineFixture(t *testing.T, dirMode, binMode os.FileMode) string {
	t.Helper()

	dir := t.TempDir()
	bin := filepath.Join(dir, "keepassxc-cli")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0755))
	require.NoError(t, os.Chmod(bin, binMode))
	require.NoError(t, os.Chmod(dir, dirMode))

	return bin
}

func TestCheckEngineFileAcceptsPrivateBinary(t *testing.T) {
	bin := engineFixture(t, 0755, 0755)
	assert.NoError(t, checkEngineFile(bin))
}

func TestCheckEngineFileRefusesGroupWritableBinary(t *testing.T) {
	bin := engineFixture(t, 0755, 0775)
	assert.Error(t, checkEngineFile(bin))
}

func TestCheckEngineFileRefusesWorldWritableDir(t *testing.T) {
	bin := engineFixture(t, 0777, 0755)
	assert.Error(t, checkEngineFile(bin))
}

func TestCheckEngineFileRefusesLinkInWorldWritableDir(t *testing.T) {
	bin := engineFixture(t, 0755, 0755)
	linkDir := t.TempDir()
	require.NoError(t, os.Chmod(linkDir, 0777))
	link := filepath.Join(linkDir, "keepassxc-cli")
	require.NoError(t, os.Symlink(bin, link))

	assert.Error(t, checkEngineFile(link))
}

func TestCheckEngineFileAcceptsSystemBinary(t *testing.T) {
	for _, p := range []string{"/usr/bin/keepassxc-cli", "/bin/sh"} {
		if _, err := os.Stat(p); err == nil {
			assert.NoError(t, checkEngineFile(p))
			return
		}
	}
	t.Skip("no system binary to check")
}
