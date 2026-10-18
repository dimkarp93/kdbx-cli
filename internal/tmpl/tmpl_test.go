package tmpl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestTitlesExtractsUnique(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"user":"{{User}}","password":"{{AdminPw}}","again":"{{User}}"}`)

	got, err := Titles(path)

	require.NoError(t, err)
	assert.Equal(t, []string{"User", "AdminPw"}, got)
}

func TestRenderRaw(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.env", "PASSWORD={{AdminPw}}\n")

	out, err := Render(path, map[string]string{"AdminPw": "s3cret"})

	require.NoError(t, err)
	assert.Equal(t, "PASSWORD=s3cret\n", string(out))
}

func TestRenderJSONEscaping(t *testing.T) {
	cases := map[string]string{
		"settings.json":      `weird"va\lue` + "\nwith newline",
		"settings.json.tmpl": `has "quotes" and` + "\nnewline",
	}

	for name, secret := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTemplate(t, t.TempDir(), name, `{"user":"Bob","password":"{{AdminPw}}"}`)
			out, err := Render(path, map[string]string{"AdminPw": secret})
			require.NoError(t, err)

			var decoded map[string]string
			require.NoError(t, json.Unmarshal(out, &decoded), "rendered file is not valid JSON:\n%s", out)

			assert.Equal(t, secret, decoded["password"])
		})
	}
}

func TestRenderMissingSecret(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"password":"{{AdminPw}}"}`)

	_, err := Render(path, map[string]string{})
	assert.ErrorContains(t, err, "AdminPw")
}
