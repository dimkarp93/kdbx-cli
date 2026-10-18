package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

func writeTestFile(t *testing.T, path, content string) string {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func captureStdout(t *testing.T, print func()) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	printTo(f, print)

	out, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	return string(out)
}

func printTo(f *os.File, print func()) {
	orig := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = orig }()

	print()
}

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
			assert.Equal(t, c.wantLeft, left)
			assert.Equal(t, c.wantChild, child)
			assert.Equal(t, c.wantSep, sep)
		})
	}
}

func TestParseRunFlags(t *testing.T) {
	f, err := parseRunFlags([]string{"--config", "/c", "--key-store=/k", "--secrets", "A:AA,B:BB", "--secrets=C:CC"})

	t.Run("succeeds", func(t *testing.T) {
		require.NoError(t, err)
	})
	t.Run("config path", func(t *testing.T) {
		assert.Equal(t, "/c", f.configPath)
	})
	t.Run("key store", func(t *testing.T) {
		assert.Equal(t, "/k", f.keyStore)
	})
	t.Run("secrets merged from repeated flags", func(t *testing.T) {
		assert.Equal(t, map[string]string{"AA": "A", "BB": "B", "CC": "C"}, f.secrets)
	})
}

func TestParseRunFlagsDryRun(t *testing.T) {
	f, err := parseRunFlags([]string{"--dry-run", "--key-store=/k"})
	require.NoError(t, err)
	assert.True(t, f.dryRun)
}

func TestParseRunFlagsDryRunDefaultsToFalse(t *testing.T) {
	f, err := parseRunFlags([]string{"--key-store=/k"})
	require.NoError(t, err)
	assert.False(t, f.dryRun)
}

func TestParseRunFlagsErrors(t *testing.T) {
	cases := map[string][]string{
		"missing value":      {"--config"},
		"secret no colon":    {"--secrets", "noColon"},
		"secret empty name":  {"--secrets", ":env"},
		"secret empty env":   {"--secrets", "name:"},
		"unknown flag":       {"--unknown"},
		"askpass reads zero": {"--askpass-reads=0"},
		"askpass reads neg":  {"--askpass-reads=-1"},
		"askpass reads text": {"--askpass-reads=x"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parseRunFlags(args)
			assert.Error(t, err)
		})
	}
}

func TestParseRunFlagsTemplate(t *testing.T) {
	f, err := parseRunFlags([]string{"--template", "cfg:/a/settings.json", "--template=cfg2:/b/other.env"})
	require.NoError(t, err)
	assert.Equal(t, []config.Template{{Name: "cfg", Path: "/a/settings.json"}, {Name: "cfg2", Path: "/b/other.env"}}, f.templates)
}

func TestParseTemplatesFlagErrors(t *testing.T) {
	for _, value := range []string{"noColon", ":path", "name:"} {
		t.Run(value, func(t *testing.T) {
			_, err := parseTemplatesFlag(value)
			assert.Error(t, err)
		})
	}
}

func TestParseRunFlagsDeliveryModes(t *testing.T) {
	f, err := parseRunFlags([]string{
		"--stdin", "A,B", "--stdin=C", "--stdin-keep-open",
		"--secret-file=F1, F2", "--secret-file", "F3",
		"--askpass=P",
	})

	t.Run("succeeds", func(t *testing.T) {
		require.NoError(t, err)
	})
	t.Run("stdin accumulates", func(t *testing.T) {
		assert.Equal(t, []string{"A", "B", "C"}, f.stdin)
	})
	t.Run("stdin keep open", func(t *testing.T) {
		assert.True(t, f.stdinKeepOpen)
	})
	t.Run("files trimmed and accumulated", func(t *testing.T) {
		assert.Equal(t, []string{"F1", "F2", "F3"}, f.files)
	})
	t.Run("askpass", func(t *testing.T) {
		assert.Equal(t, "P", f.askpass)
	})
}

func TestParseRunFlagsAskpassReads(t *testing.T) {
	f, err := parseRunFlags([]string{"--askpass=PW", "--askpass-reads=3"})

	t.Run("succeeds", func(t *testing.T) {
		require.NoError(t, err)
	})
	t.Run("parsed", func(t *testing.T) {
		assert.Equal(t, 3, f.askpassReads)
	})
	t.Run("passed to overrides", func(t *testing.T) {
		assert.Equal(t, 3, f.overrides().AskpassReads)
	})
}

func TestParseRunFlagsAskpassReadsSeparateValue(t *testing.T) {
	f, err := parseRunFlags([]string{"--askpass-reads", "2"})
	require.NoError(t, err)
	assert.Equal(t, 2, f.askpassReads)
}

func TestMergeSecretsFlag(t *testing.T) {
	m := map[string]string{}

	err := mergeSecretsFlag(m, "Group/Sub/Title:ENV, X:Y , Group/Sub/Title:ENV2")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"ENV": "Group/Sub/Title", "ENV2": "Group/Sub/Title", "Y": "X"}, m)
}

func TestDescribeSection(t *testing.T) {
	withDefault := config.Config{Sections: map[string]config.Section{"default": {}, "install": {}}}

	cases := []struct {
		name string
		cfg  config.Config
		tool string
		want string
	}{
		{"tool over default", withDefault, "install", `"install" (merged over "default")`},
		{"no tool section", withDefault, "curl", `"default" (no section for "curl")`},
		{"default itself", withDefault, "default", `"default"`},
		{"empty config", config.Config{}, "x", `(no section for "x" and no "default")`},
		{"tool section only", config.Config{Sections: map[string]config.Section{"only": {}}}, "only", `"only"`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, describeSection(c.cfg, c.tool))
		})
	}
}

func TestRenderCommand(t *testing.T) {
	secrets := map[string]string{"GH_TOKEN": "GITHUB_TOKEN", "NPM_TOKEN": "NPM_TOKEN"}

	got := renderCommand(sortedMappings(secrets), nil, []string{"install", "arg with spaces", "plain"})
	assert.Equal(t, "GH_TOKEN=<secret from GITHUB_TOKEN> NPM_TOKEN=<secret from NPM_TOKEN> install 'arg with spaces' plain", got)
}

func TestRenderCommandWithFiles(t *testing.T) {
	got := renderCommand(nil, []string{"repo"}, []string{"restic", "--password-file", "{{repo}}"})
	assert.Equal(t, "restic --password-file '<file with repo>'", got)
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
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, shellQuote(in))
		})
	}
}

func TestPrintPlanShowsTemplatesBlock(t *testing.T) {
	tmplPath := writeTestFile(t, filepath.Join(t.TempDir(), "settings.json"), `{"password":"{{AdminPw}}"}`)
	res := resolve.Resolved{Templates: []config.Template{{Name: "cfg", Path: tmplPath}}}

	out := captureStdout(t, func() {
		printPlan(runPlan{cfgPath: "/cfg", tool: "admin", res: res}, runFlags{}, []string{"admin", "{{cfg}}"})
	})

	t.Run("templates block", func(t *testing.T) {
		assert.Contains(t, out, "Templates (placeholder ← rendered file):")
	})
	t.Run("template titles", func(t *testing.T) {
		assert.Contains(t, out, "(uses: AdminPw)")
	})
	t.Run("command masks template file", func(t *testing.T) {
		assert.Contains(t, out, "<file with cfg>")
	})
}

func TestTitlesGranted(t *testing.T) {
	granted := map[string]bool{"A": true, "B": true}

	cases := []struct {
		name   string
		titles []string
		want   bool
	}{
		{"subset", []string{"A", "B"}, true},
		{"empty", nil, true},
		{"unknown title", []string{"A", "C"}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, titlesGranted(c.titles, granted))
		})
	}
}
