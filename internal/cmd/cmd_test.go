package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/domain"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
)

func TestSplitArgs(t *testing.T) {
	cases := []struct {
		name      string
		in        []string
		wantLeft  []string
		wantChild []string
		wantSep   bool
	}{
		{"no sep", []string{"--config", "x"}, []string{"--config", "x"}, nil, false},
		{"sep at end", []string{"--config", "x", "--"}, []string{"--config", "x"}, []string{}, true},
		{"basic", []string{"--config", "x", "--", "install", "a/b"}, []string{"--config", "x"}, []string{"install", "a/b"}, true},
		{"sep first", []string{"--", "tool"}, []string{}, []string{"tool"}, true},
		{"child has flags", []string{"--", "tool", "-v", "--x"}, []string{}, []string{"tool", "-v", "--x"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			left, child, sep := splitArgs(c.in)
			if !reflect.DeepEqual(left, c.wantLeft) {
				t.Errorf("left: got %v, want %v", left, c.wantLeft)
			}
			if !reflect.DeepEqual(child, c.wantChild) {
				t.Errorf("child: got %v, want %v", child, c.wantChild)
			}
			if sep != c.wantSep {
				t.Errorf("sep: got %v, want %v", sep, c.wantSep)
			}
		})
	}
}

func TestParseRunFlags(t *testing.T) {
	f, err := parseRunFlags([]string{"--config", "/c", "--key-store=/k", "--secrets", "A:AA,B:BB", "--secrets=C:CC"})
	if err != nil {
		t.Fatal(err)
	}
	if f.configPath != "/c" {
		t.Errorf("configPath: got %q", f.configPath)
	}
	if f.keyStore != "/k" {
		t.Errorf("keyStore: got %q", f.keyStore)
	}
	want := map[string]string{"A": "AA", "B": "BB", "C": "CC"}
	if !reflect.DeepEqual(f.secrets, want) {
		t.Errorf("secrets: got %v, want %v", f.secrets, want)
	}
}

func TestParseRunFlagsDryRun(t *testing.T) {
	f, err := parseRunFlags([]string{"--dry-run", "--key-store=/k"})
	if err != nil {
		t.Fatal(err)
	}
	if !f.dryRun {
		t.Error("dryRun should be true")
	}
	f2, _ := parseRunFlags([]string{"--key-store=/k"})
	if f2.dryRun {
		t.Error("dryRun should default to false")
	}
}

func TestParseRunFlagsErrors(t *testing.T) {
	cases := [][]string{
		{"--config"},
		{"--secrets", "noColon"},
		{"--secrets", ":env"},
		{"--secrets", "name:"},
		{"--unknown"},
	}
	for _, c := range cases {
		if _, err := parseRunFlags(c); err == nil {
			t.Errorf("expected error for %v", c)
		}
	}
}

func TestMergeSecretsFlag(t *testing.T) {
	m := map[string]string{}
	if err := mergeSecretsFlag(m, "Group/Sub/Title:ENV, X:Y ,"); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Group/Sub/Title": "ENV", "X": "Y"}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %v, want %v", m, want)
	}
}

func TestDescribeSection(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{"default": {}, "install": {}}}
	cases := map[string]string{
		"install": `"install" (merged over "default")`,
		"curl":    `"default" (no section for "curl")`,
		"default": `"default"`,
	}
	for tool, want := range cases {
		if got := describeSection(cfg, tool); got != want {
			t.Errorf("describeSection(%q): got %q, want %q", tool, got, want)
		}
	}
	if got := describeSection(config.Config{}, "x"); got != `(no section for "x" and no "default")` {
		t.Errorf("empty config: got %q", got)
	}
	if got := describeSection(config.Config{Sections: map[string]config.Section{"only": {}}}, "only"); got != `"only"` {
		t.Errorf("tool-only section: got %q", got)
	}
}

func TestRenderCommand(t *testing.T) {
	secrets := map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "NPM_TOKEN": "NPM_TOKEN"}
	got := renderCommand(sortedMappings(secrets), nil, []string{"install", "arg with spaces", "plain"})
	want := "GH_TOKEN=<secret from GITHUB_TOKEN> NPM_TOKEN=<secret from NPM_TOKEN> install 'arg with spaces' plain"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"plain":         "plain",
		"a/b-c.d":       "a/b-c.d",
		"with space":    "'with space'",
		"":              "''",
		"it's":          `'it'\''s'`,
		"https://x?a=1": "'https://x?a=1'",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q): got %q, want %q", in, got, want)
		}
	}
}

func TestTitlesOf(t *testing.T) {
	got := titlesOf([]domain.Mapping{{Name: "B", Env: "2"}, {Name: "A", Env: "1"}})
	if !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("got %v", got)
	}
}

func TestRefreshPathSuggestions(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "store.kdbx"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}

	ti := textinput.New()
	ti.SetValue(dir + string(os.PathSeparator))
	refreshPathSuggestions(&ti)
	got := ti.AvailableSuggestions()

	wantFile := filepath.Join(dir, "store.kdbx")
	wantDir := filepath.Join(dir, "sub") + string(os.PathSeparator)
	if !slices.Contains(got, wantFile) {
		t.Errorf("suggestions missing file %q: %v", wantFile, got)
	}
	if !slices.Contains(got, wantDir) {
		t.Errorf("suggestions missing dir %q: %v", wantDir, got)
	}
}

func TestCommitEditAddSortAndDuplicate(t *testing.T) {
	m := newConfigTUI("", []domain.Mapping{{Name: "B", Env: "2"}}, nil)
	m.mode = modeEdit
	m.editIndex = -1
	m.name.SetValue("A")
	m.env.SetValue("1")
	res, _ := m.commitEdit()
	m = res.(configTUI)
	if len(m.pairs) != 2 || m.pairs[0].Name != "A" {
		t.Fatalf("add/sort failed: %v", m.pairs)
	}
	if m.cursor != 0 {
		t.Errorf("cursor should point at the committed item, got %d", m.cursor)
	}

	m.mode = modeEdit
	m.editIndex = -1
	m.name.SetValue("B")
	m.env.SetValue("9")
	res, _ = m.commitEdit()
	dup := res.(configTUI)
	if dup.err == "" || dup.mode != modeEdit {
		t.Errorf("expected duplicate error and stay in edit, err=%q mode=%v", dup.err, dup.mode)
	}
}

func TestCommitEditRequiresBothFields(t *testing.T) {
	m := newConfigTUI("", nil, nil)
	m.mode = modeEdit
	m.editIndex = -1
	m.name.SetValue("")
	m.env.SetValue("X")
	res, _ := m.commitEdit()
	got := res.(configTUI)
	if got.err == "" || len(got.pairs) != 0 {
		t.Errorf("empty name should error and add nothing, err=%q pairs=%v", got.err, got.pairs)
	}
}

func TestListDeleteAdjustsCursor(t *testing.T) {
	m := newConfigTUI("", []domain.Mapping{{Name: "A", Env: "1"}, {Name: "B", Env: "2"}}, nil)
	m.mode = modeList
	m.cursor = 1
	res, _ := m.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = res.(configTUI)
	if len(m.pairs) != 1 || m.pairs[0].Name != "A" {
		t.Fatalf("delete failed: %v", m.pairs)
	}
	if m.cursor != 0 {
		t.Errorf("cursor not clamped after delete: %d", m.cursor)
	}
}

func TestDeleteLastPathSegment(t *testing.T) {
	cases := map[string]string{
		"/home/dima/vault/store.kdbx": "/home/dima/vault/",
		"/home/dima/vault/":           "/home/dima/",
		"/home/":                      "/",
		"/":                           "",
		"relative/path":               "relative/",
		"single":                      "",
	}
	for in, want := range cases {
		ti := textinput.New()
		ti.SetValue(in)
		deleteLastPathSegment(&ti)
		if got := ti.Value(); got != want {
			t.Errorf("deleteLastPathSegment(%q): got %q, want %q", in, got, want)
		}
	}
}

func TestShowTUINavigationAndOpen(t *testing.T) {
	views := []domain.StoreView{
		{Path: "/a.kdbx", Exists: true},
		{Path: "/b.kdbx", Exists: false},
	}
	m := newShowTUI("/cfg", nil, views, keyring.New(nil))

	down := tea.KeyMsg{Type: tea.KeyDown}
	res, _ := m.Update(down)
	m = res.(showTUI)
	if m.cursor != 1 {
		t.Fatalf("cursor after down: %d", m.cursor)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(showTUI)
	if !strings.Contains(m.status, "does not exist") {
		t.Errorf("opening missing file should report it, status=%q", m.status)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(showTUI)
	if m.cursor != 0 {
		t.Fatalf("cursor after up: %d", m.cursor)
	}

	called := ""
	orig := openInKeePassXC
	openInKeePassXC = func(p string) error { called = p; return nil }
	defer func() { openInKeePassXC = orig }()

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(showTUI)
	if called != "/a.kdbx" {
		t.Errorf("expected open of /a.kdbx, got %q", called)
	}
	if !strings.Contains(m.status, "opened in KeePassXC") {
		t.Errorf("status after open: %q", m.status)
	}
}

func TestParseRunFlagsDeliveryModes(t *testing.T) {
	f, err := parseRunFlags([]string{
		"--stdin", "A,B", "--stdin=C", "--stdin-keep-open",
		"--secret-file=F1, F2", "--secret-file", "F3",
		"--askpass=P",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.stdin, []string{"A", "B", "C"}) {
		t.Errorf("stdin: got %v", f.stdin)
	}
	if !f.stdinKeepOpen {
		t.Error("stdinKeepOpen should be true")
	}
	if !reflect.DeepEqual(f.files, []string{"F1", "F2", "F3"}) {
		t.Errorf("files: got %v", f.files)
	}
	if f.askpass != "P" {
		t.Errorf("askpass: got %q", f.askpass)
	}
}

func TestStdinPayload(t *testing.T) {
	got, err := stdinPayload([]string{"pw", "pw"}, map[string]string{"pw": "s3cret"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "s3cret\ns3cret\n" {
		t.Errorf("got %q", got)
	}
	if _, err := stdinPayload([]string{"pw"}, map[string]string{"pw": "a\nb"}); err == nil {
		t.Error("expected an error for a secret with a newline")
	}
}

func TestSubstitutePlaceholders(t *testing.T) {
	args := []string{"restic", "--password-file", "{{repo pw}}", "--other=x{{repo pw}}x"}
	got, err := substitutePlaceholders(args, map[string]string{"repo pw": "/run/fifo"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"restic", "--password-file", "/run/fifo", "--other=x/run/fifox"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if args[2] != "{{repo pw}}" {
		t.Error("input args were mutated")
	}
	if _, err := substitutePlaceholders([]string{"restic"}, map[string]string{"pw": "/run/fifo"}); err == nil {
		t.Error("expected an error when the placeholder is absent")
	}
}

func TestWithAskpass(t *testing.T) {
	env := withAskpass([]string{"PATH=/bin", "SSH_ASKPASS=/old", "HOME=/h"}, "/tmp/askpass.sh")
	if slices.Contains(env, "SSH_ASKPASS=/old") {
		t.Error("the pre-existing SSH_ASKPASS was not replaced")
	}
	for _, want := range []string{
		"PATH=/bin", "HOME=/h",
		"SSH_ASKPASS=/tmp/askpass.sh", "SUDO_ASKPASS=/tmp/askpass.sh",
		"GIT_ASKPASS=/tmp/askpass.sh", "RESTIC_PASSWORD_COMMAND=/tmp/askpass.sh",
		"BORG_PASSCOMMAND=/tmp/askpass.sh", "SSH_ASKPASS_REQUIRE=force",
		"GIT_TERMINAL_PROMPT=0",
	} {
		if !slices.Contains(env, want) {
			t.Errorf("env missing %q: %v", want, env)
		}
	}
}

func TestRenderCommandWithFiles(t *testing.T) {
	got := renderCommand(nil, []string{"repo"}, []string{"restic", "--password-file", "{{repo}}"})
	want := "restic --password-file '<file with repo>'"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
