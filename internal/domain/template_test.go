package domain

import (
	"encoding/json"
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

func TestTemplateTitlesExtractsUnique(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"user":"{{User}}","password":"{{AdminPw}}","again":"{{User}}"}`)

	got, err := TemplateTitles(path)

	require.NoError(t, err)
	assert.Equal(t, []string{"User", "AdminPw"}, got)
}

func TestRenderTemplateRaw(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.env", "PASSWORD={{AdminPw}}\n")

	out, err := RenderTemplate(path, map[string]string{"AdminPw": "s3cret"})

	require.NoError(t, err)
	assert.Equal(t, "PASSWORD=s3cret\n", string(out))
}

func TestRenderTemplateJSONEscaping(t *testing.T) {
	cases := map[string]string{
		"settings.json":      `weird"va\lue` + "\nwith newline",
		"settings.json.tmpl": `has "quotes" and` + "\nnewline",
	}

	for name, secret := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTemplate(t, t.TempDir(), name, `{"user":"Bob","password":"{{AdminPw}}"}`)
			out, err := RenderTemplate(path, map[string]string{"AdminPw": secret})
			require.NoError(t, err)

			var decoded map[string]string
			require.NoError(t, json.Unmarshal(out, &decoded), "rendered file is not valid JSON:\n%s", out)

			assert.Equal(t, secret, decoded["password"])
		})
	}
}

func TestRenderTemplateMissingSecret(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"password":"{{AdminPw}}"}`)

	_, err := RenderTemplate(path, map[string]string{})
	assert.ErrorContains(t, err, "AdminPw")
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
