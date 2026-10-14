package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/domain"
)

func TestLoadRunConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent")
	gotPath, cfg, err := loadRunConfig(path)
	if err != nil {
		t.Fatalf("missing config must not fail: %v", err)
	}
	if gotPath != path || len(cfg.Sections) != 0 {
		t.Errorf("path=%q cfg=%+v", gotPath, cfg)
	}
}

func TestLoadRunConfigInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad")
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadRunConfig(path); err == nil {
		t.Error("invalid config must fail")
	}
}

func TestRequestedTitlesMergesChannels(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "t.txt")
	if err := os.WriteFile(tmpl, []byte("{{C}} and {{A}}"), 0600); err != nil {
		t.Fatal(err)
	}
	res := domain.Resolved{
		Secrets:   map[string]string{"X": "A"},
		Stdin:     []string{"B"},
		Templates: []config.Template{{Name: "t", Path: tmpl}},
	}
	got, err := requestedTitles(res)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"A", "B", "C"}) {
		t.Errorf("titles: %v", got)
	}
}

func TestRequestedTitlesTemplateError(t *testing.T) {
	res := domain.Resolved{Templates: []config.Template{{Name: "t", Path: filepath.Join(t.TempDir(), "absent")}}}
	if _, err := requestedTitles(res); err == nil {
		t.Error("unreadable template must fail")
	}
}

func TestCmdLoadRejectsDryRun(t *testing.T) {
	if code := cmdLoad(runFlags{dryRun: true}); code != 2 {
		t.Errorf("exit code: %d", code)
	}
}
