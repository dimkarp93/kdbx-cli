package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestResolveMerge(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "~/store.kdbx", Secrets: map[string]string{"GH_TOKEN": "GITHUB_TOKEN", "SHARED_ENV": "SHARED"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM_TOKEN", "SHARED_ENV": "OVERRIDE", "SHARED_COPY": "SHARED"}},
	}}

	r := Resolve(cfg, "install", Overrides{})

	t.Run("key store inherited from default", func(t *testing.T) {
		assert.Equal(t, "~/store.kdbx", r.KeyStore)
	})
	t.Run("secrets merged per entry", func(t *testing.T) {
		want := map[string]string{"GH_TOKEN": "GITHUB_TOKEN", "SHARED_ENV": "OVERRIDE", "SHARED_COPY": "SHARED", "NPM_TOKEN": "NPM_TOKEN"}
		assert.Equal(t, want, r.Secrets)
	})
}

func TestResolveFlagsOverride(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/from-cfg", Secrets: map[string]string{"E1": "A"}},
	}}

	r := Resolve(cfg, "unknown-tool", Overrides{KeyStore: "/from-flag", Secrets: map[string]string{"E1": "A2", "E2": "B"}})

	t.Run("key store", func(t *testing.T) {
		assert.Equal(t, "/from-flag", r.KeyStore)
	})
	t.Run("secrets", func(t *testing.T) {
		assert.Equal(t, map[string]string{"E1": "A2", "E2": "B"}, r.Secrets)
	})
}

func deliveryConfig() config.Config {
	return config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s", Stdin: []string{"D1"}, Files: []string{"F1"}, Templates: []config.Template{{Name: "T1", Path: "/t1"}}, Askpass: "A1"},
		"docker":  {Stdin: []string{"D2", "D3"}},
	}}
}

func TestResolveDeliveryModesFromSections(t *testing.T) {
	r := Resolve(deliveryConfig(), "docker", Overrides{})

	t.Run("stdin replaced not merged", func(t *testing.T) {
		assert.Equal(t, []string{"D2", "D3"}, r.Stdin)
	})
	t.Run("files inherited", func(t *testing.T) {
		assert.Equal(t, []string{"F1"}, r.Files)
	})
	t.Run("templates inherited", func(t *testing.T) {
		assert.Equal(t, []config.Template{{Name: "T1", Path: "/t1"}}, r.Templates)
	})
	t.Run("askpass inherited", func(t *testing.T) {
		assert.Equal(t, "A1", r.Askpass)
	})
}

func TestResolveDeliveryModesFromFlags(t *testing.T) {
	flags := Overrides{Stdin: []string{"D4"}, StdinKeepOpen: true, Templates: []config.Template{{Name: "T2", Path: "/t2"}}, Askpass: "A2"}

	r := Resolve(deliveryConfig(), "docker", flags)

	t.Run("stdin", func(t *testing.T) {
		assert.Equal(t, []string{"D4"}, r.Stdin)
	})
	t.Run("stdin keep open", func(t *testing.T) {
		assert.True(t, r.StdinKeepOpen)
	})
	t.Run("templates", func(t *testing.T) {
		assert.Equal(t, []config.Template{{Name: "T2", Path: "/t2"}}, r.Templates)
	})
	t.Run("askpass", func(t *testing.T) {
		assert.Equal(t, "A2", r.Askpass)
	})
}

func TestResolveAskpassReads(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {AskpassReads: 2},
		"restic":  {AskpassReads: 4},
	}}

	cases := []struct {
		name  string
		tool  string
		flags Overrides
		want  int
	}{
		{"default section", "git", Overrides{}, 2},
		{"tool section", "restic", Overrides{}, 4},
		{"flag", "restic", Overrides{AskpassReads: 7}, 7},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, Resolve(cfg, c.tool, c.flags).AskpassReads)
		})
	}
}

func TestMappingsFromMapSortsByName(t *testing.T) {
	got := MappingsFromMap(map[string]string{"2": "B", "1": "A", "3": "C", "4": "A"})
	want := []Mapping{{Name: "A", Env: "1"}, {Name: "A", Env: "4"}, {Name: "B", Env: "2"}, {Name: "C", Env: "3"}}
	assert.Equal(t, want, got)
}

func TestAllTitlesDeduplicates(t *testing.T) {
	r := Resolved{
		Secrets: map[string]string{"ENV": "T", "ENV2": "T"},
		Stdin:   []string{"T", "S"},
		Files:   []string{"S", "F"},
		Askpass: "F",
	}

	assert.ElementsMatch(t, []string{"F", "S", "T"}, r.AllTitles())
}

func TestValidateEnvNamesAcceptsValid(t *testing.T) {
	assert.NoError(t, ValidateEnvNames(map[string]string{"GOOD_1": "a", "_ok": "b"}))
}

func TestValidateEnvNamesRejectsInvalid(t *testing.T) {
	err := ValidateEnvNames(map[string]string{"1BAD": "a", "A=B": "b", "": "c", "OK": "d"})

	t.Run("fails", func(t *testing.T) {
		require.Error(t, err)
	})
	for _, name := range []string{`"1BAD"`, `"A=B"`, `""`} {
		t.Run("names "+name, func(t *testing.T) {
			assert.ErrorContains(t, err, name)
		})
	}
	t.Run("omits valid name", func(t *testing.T) {
		require.Error(t, err)
		assert.NotContains(t, err.Error(), `"OK"`)
	})
}

func TestConfiguredTitles(t *testing.T) {
	tmpl := writeTemplate(t, t.TempDir(), "conf.tmpl", "token={{TMPL}}")
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {Secrets: map[string]string{"A": "DEF"}},
		"tool":    {Stdin: []string{"IN"}, Askpass: "ASK", Templates: []config.Template{{Name: "c", Path: tmpl}}},
		"other":   {Secrets: map[string]string{"B": "OTHER"}},
	}}

	got := ConfiguredTitles(cfg, "tool")

	for _, name := range []string{"DEF", "IN", "ASK", "TMPL"} {
		t.Run("grants "+name, func(t *testing.T) {
			assert.Contains(t, got, name)
		})
	}
	t.Run("skips other sections", func(t *testing.T) {
		assert.NotContains(t, got, "OTHER")
	})
}

func TestResolvedTemplateTitlesEmpty(t *testing.T) {
	titles, err := (Resolved{}).TemplateTitles()
	require.NoError(t, err)
	assert.Empty(t, titles)
}

func TestResolvedTemplateTitlesDedup(t *testing.T) {
	dir := t.TempDir()
	p1 := writeTemplate(t, dir, "a.json", `{"a":"{{Shared}}"}`)
	p2 := writeTemplate(t, dir, "b.json", `{"b":"{{Shared}}","c":"{{Other}}"}`)
	r := Resolved{Templates: []config.Template{{Name: "t1", Path: p1}, {Name: "t2", Path: p2}}}

	got, err := r.TemplateTitles()

	require.NoError(t, err)
	assert.Equal(t, []string{"Shared", "Other"}, got)
}
