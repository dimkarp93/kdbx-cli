package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestAggregateStores(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/a.kdbx", Secrets: map[string]string{"GH": "GITHUB_TOKEN", "GH_COPY": "GITHUB_TOKEN"}},
		"install": {Secrets: map[string]string{"NPM": "NPM_TOKEN"}},
		"deploy":  {KeyStore: "/b.kdbx", Secrets: map[string]string{"AWS": "AWS_KEY"}},
	}}

	got := AggregateStores(cfg)

	t.Run("default store collects inheriting sections", func(t *testing.T) {
		assert.Equal(t, []string{"GITHUB_TOKEN", "NPM_TOKEN"}, got["/a.kdbx"])
	})
	t.Run("own store includes default secrets", func(t *testing.T) {
		assert.Equal(t, []string{"AWS_KEY", "GITHUB_TOKEN"}, got["/b.kdbx"])
	})
}

func TestAggregateStoresCoversAllChannels(t *testing.T) {
	tmplPath := writeTemplate(t, t.TempDir(), "settings.json.tmpl", `{"password":"{{T}}"}`)
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s"},
		"docker":  {Stdin: []string{"D"}},
		"restic":  {Files: []string{"F"}},
		"admin":   {Templates: []config.Template{{Name: "cfg", Path: tmplPath}}},
		"ssh":     {Askpass: "A"},
	}}

	assert.Equal(t, []string{"A", "D", "F", "T"}, AggregateStores(cfg)["/s"])
}

func TestAggregateStoreMappingsChannelTemplate(t *testing.T) {
	tmplPath := writeTemplate(t, t.TempDir(), "settings.json.tmpl", `{"password":"{{AdminPw}}"}`)
	cfg := config.Config{Sections: map[string]config.Section{
		"admin": {KeyStore: "/s", Templates: []config.Template{{Name: "cfg", Path: tmplPath}}},
	}}

	assert.Equal(t, []resolve.Mapping{{Name: "AdminPw", Env: ChannelTemplate}}, AggregateStoreMappings(cfg)["/s"])
}

func TestAggregateStoreMappingsMultipleEnvPerSecret(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s", Secrets: map[string]string{"A_ENV": "T", "B_ENV": "T"}, Stdin: []string{"T", "U"}},
	}}

	want := []resolve.Mapping{{Name: "T", Env: "A_ENV"}, {Name: "T", Env: "B_ENV"}, {Name: "U", Env: ChannelStdin}}
	assert.Equal(t, want, AggregateStoreMappings(cfg)["/s"])
}

func TestBuildStoreViewsMarksExistence(t *testing.T) {
	dir := t.TempDir()
	existing := writeTemplate(t, dir, "store.kdbx", "")
	missing := filepath.Join(dir, "absent.kdbx")
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: existing, Secrets: map[string]string{"GH": "GITHUB_TOKEN"}},
		"deploy":  {KeyStore: missing, Secrets: map[string]string{"AWS": "AWS_KEY"}},
	}}

	want := []StoreView{
		{Path: missing, Exists: false, Mappings: []resolve.Mapping{{Name: "AWS_KEY", Env: "AWS"}, {Name: "GITHUB_TOKEN", Env: "GH"}}},
		{Path: existing, Exists: true, Mappings: []resolve.Mapping{{Name: "GITHUB_TOKEN", Env: "GH"}}},
	}
	assert.Equal(t, want, BuildStoreViews(cfg))
}

type scriptedAnswers struct {
	answers []string
}

func (s *scriptedAnswers) read(string) (string, error) {
	if len(s.answers) == 0 {
		return "", errors.New("no more answers")
	}

	answer := s.answers[0]
	s.answers = s.answers[1:]
	return answer, nil
}

func TestReadNewPasswordMatching(t *testing.T) {
	answers := &scriptedAnswers{answers: []string{"pw", "pw"}}

	pw, err := readNewPassword("/k.kdbx", answers.read)

	require.NoError(t, err)
	assert.Equal(t, "pw", pw)
}

func TestReadNewPasswordMismatch(t *testing.T) {
	answers := &scriptedAnswers{answers: []string{"pw", "typo"}}

	_, err := readNewPassword("/k.kdbx", answers.read)
	assert.Error(t, err)
}

func TestReadNewPasswordEmpty(t *testing.T) {
	answers := &scriptedAnswers{answers: []string{""}}

	_, err := readNewPassword("/k.kdbx", answers.read)
	assert.Error(t, err)
}
