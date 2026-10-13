package keepass

import (
	"os"
	"path/filepath"
	"testing"
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

func TestParseSecrets(t *testing.T) {
	entries, err := ParseSecrets(sampleXML)
	if err != nil {
		t.Fatal(err)
	}

	v, err := LookupSecret(entries, "GITHUB_TOKEN")
	if err != nil || v != "ghp_top" {
		t.Errorf("GITHUB_TOKEN: got %q, err %v", v, err)
	}

	v, err = LookupSecret(entries, "API_KEY")
	if err != nil || v != "web_secret" {
		t.Errorf("API_KEY: got %q, err %v", v, err)
	}

	v, err = LookupSecret(entries, "web/API_KEY")
	if err != nil || v != "web_secret" {
		t.Errorf("web/API_KEY by path: got %q, err %v", v, err)
	}

	if _, err := LookupSecret(entries, "DUP"); err == nil {
		t.Error("expected ambiguity error for DUP")
	}

	v, err = LookupSecret(entries, "db/DUP")
	if err != nil || v != "db_dup" {
		t.Errorf("db/DUP by path: got %q, err %v", v, err)
	}

	if _, err := LookupSecret(entries, "MISSING"); err == nil {
		t.Error("expected not-found error for MISSING")
	}
}

func TestCheckEngineFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "keepassxc-cli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := checkEngineFile(bin); err != nil {
		t.Errorf("private binary should pass: %v", err)
	}

	if err := os.Chmod(bin, 0775); err != nil {
		t.Fatal(err)
	}
	if err := checkEngineFile(bin); err == nil {
		t.Error("group-writable binary should be refused")
	}
	if err := os.Chmod(bin, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatal(err)
	}
	if err := checkEngineFile(bin); err == nil {
		t.Error("binary in a world-writable directory should be refused")
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}

	linkDir := t.TempDir()
	if err := os.Chmod(linkDir, 0777); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "keepassxc-cli")
	if err := os.Symlink(bin, link); err != nil {
		t.Fatal(err)
	}
	if err := checkEngineFile(link); err == nil {
		t.Error("link in a world-writable PATH directory should be refused")
	}
}

func TestCheckEngineFileAcceptsSystemBinary(t *testing.T) {
	for _, p := range []string{"/usr/bin/keepassxc-cli", "/bin/sh"} {
		if _, err := os.Stat(p); err == nil {
			if err := checkEngineFile(p); err != nil {
				t.Errorf("%s: %v", p, err)
			}
			return
		}
	}
	t.Skip("no system binary to check")
}
