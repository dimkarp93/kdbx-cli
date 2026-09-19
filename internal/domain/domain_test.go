package domain

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func TestResolveMerge(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "~/store.kdbx", Secrets: map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "SHARED": "SHARED_ENV"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM_TOKEN", "SHARED": "OVERRIDE"}},
	}}

	r := Resolve(cfg, "install", Overrides{})
	if r.KeyStore != "~/store.kdbx" {
		t.Errorf("keyStore: got %q", r.KeyStore)
	}
	want := map[string]string{"GITHUB_TOKEN": "GH_TOKEN", "SHARED": "OVERRIDE", "NPM_TOKEN": "NPM_TOKEN"}
	if !reflect.DeepEqual(r.Secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.Secrets, want)
	}
}

func TestResolveFlagsOverride(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/from-cfg", Secrets: map[string]string{"A": "A1"}},
	}}
	r := Resolve(cfg, "unknown-tool", Overrides{KeyStore: "/from-flag", Secrets: map[string]string{"A": "A2", "B": "B1"}})
	if r.KeyStore != "/from-flag" {
		t.Errorf("keyStore: got %q", r.KeyStore)
	}
	want := map[string]string{"A": "A2", "B": "B1"}
	if !reflect.DeepEqual(r.Secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.Secrets, want)
	}
}

func TestMappingsRoundTrip(t *testing.T) {
	m := map[string]string{"B": "2", "A": "1", "C": "3"}
	pairs := MappingsFromMap(m)
	if pairs[0].Name != "A" || pairs[1].Name != "B" || pairs[2].Name != "C" {
		t.Errorf("not sorted by name: %v", pairs)
	}
	if !reflect.DeepEqual(MappingsToMap(pairs), m) {
		t.Errorf("roundtrip mismatch: %v", MappingsToMap(pairs))
	}
}

func TestAggregateStores(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/a.kdbx", Secrets: map[string]string{"GITHUB_TOKEN": "GH"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM"}},
		"deploy":  {KeyStore: "/b.kdbx", Secrets: map[string]string{"AWS_KEY": "AWS"}},
	}}
	got := AggregateStores(cfg)

	wantA := []string{"GITHUB_TOKEN", "NPM_TOKEN"}
	if !reflect.DeepEqual(got["/a.kdbx"], wantA) {
		t.Errorf("/a.kdbx: got %v, want %v", got["/a.kdbx"], wantA)
	}
	wantB := []string{"AWS_KEY", "GITHUB_TOKEN"}
	if !reflect.DeepEqual(got["/b.kdbx"], wantB) {
		t.Errorf("/b.kdbx: got %v, want %v", got["/b.kdbx"], wantB)
	}
}

func TestBuildStoreViews(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "store.kdbx")
	if err := os.WriteFile(existing, nil, 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "absent.kdbx")

	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: existing, Secrets: map[string]string{"GITHUB_TOKEN": "GH"}},
		"deploy":  {KeyStore: missing, Secrets: map[string]string{"AWS_KEY": "AWS"}},
	}}
	views := BuildStoreViews(cfg)
	if len(views) != 2 {
		t.Fatalf("want 2 views, got %d: %+v", len(views), views)
	}
	byPath := map[string]StoreView{}
	for _, v := range views {
		byPath[v.Path] = v
	}
	if !byPath[existing].Exists {
		t.Errorf("existing store should be marked exists")
	}
	if byPath[missing].Exists {
		t.Errorf("missing store should be marked not exists")
	}
	if len(byPath[missing].Mappings) != 2 {
		t.Errorf("deploy store mappings: %+v", byPath[missing].Mappings)
	}
}

func TestResolveDeliveryModes(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s", Stdin: []string{"D1"}, Files: []string{"F1"}, Askpass: "A1"},
		"docker":  {Stdin: []string{"D2", "D3"}},
	}}

	r := Resolve(cfg, "docker", Overrides{})
	if !reflect.DeepEqual(r.Stdin, []string{"D2", "D3"}) {
		t.Errorf("stdin should be replaced, not merged: got %v", r.Stdin)
	}
	if !reflect.DeepEqual(r.Files, []string{"F1"}) {
		t.Errorf("files: got %v", r.Files)
	}
	if r.Askpass != "A1" {
		t.Errorf("askpass: got %q", r.Askpass)
	}

	r = Resolve(cfg, "docker", Overrides{Stdin: []string{"D4"}, StdinKeepOpen: true, Askpass: "A2"})
	if !reflect.DeepEqual(r.Stdin, []string{"D4"}) {
		t.Errorf("stdin override: got %v", r.Stdin)
	}
	if !r.StdinKeepOpen {
		t.Error("stdinKeepOpen override was lost")
	}
	if r.Askpass != "A2" {
		t.Errorf("askpass override: got %q", r.Askpass)
	}
}

func TestAllTitlesDeduplicates(t *testing.T) {
	r := Resolved{
		Secrets: map[string]string{"T": "ENV"},
		Stdin:   []string{"T", "S"},
		Files:   []string{"S", "F"},
		Askpass: "F",
	}
	got := append([]string{}, r.AllTitles()...)
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"F", "S", "T"}) {
		t.Errorf("got %v", got)
	}
}

func TestAggregateStoresCoversAllChannels(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s"},
		"docker":  {Stdin: []string{"D"}},
		"restic":  {Files: []string{"F"}},
		"ssh":     {Askpass: "A"},
	}}
	got := AggregateStores(cfg)["/s"]
	if !reflect.DeepEqual(got, []string{"A", "D", "F"}) {
		t.Errorf("got %v", got)
	}
}
