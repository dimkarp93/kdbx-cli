package domain

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func TestResolveMerge(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "~/store.kdbx", Secrets: map[string]string{"GH_TOKEN": "GITHUB_TOKEN", "SHARED_ENV": "SHARED"}},
		"install": {Secrets: map[string]string{"NPM_TOKEN": "NPM_TOKEN", "SHARED_ENV": "OVERRIDE", "SHARED_COPY": "SHARED"}},
	}}

	r := Resolve(cfg, "install", Overrides{})
	if r.KeyStore != "~/store.kdbx" {
		t.Errorf("keyStore: got %q", r.KeyStore)
	}
	want := map[string]string{"GH_TOKEN": "GITHUB_TOKEN", "SHARED_ENV": "OVERRIDE", "SHARED_COPY": "SHARED", "NPM_TOKEN": "NPM_TOKEN"}
	if !reflect.DeepEqual(r.Secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.Secrets, want)
	}
}

func TestResolveFlagsOverride(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/from-cfg", Secrets: map[string]string{"E1": "A"}},
	}}
	r := Resolve(cfg, "unknown-tool", Overrides{KeyStore: "/from-flag", Secrets: map[string]string{"E1": "A2", "E2": "B"}})
	if r.KeyStore != "/from-flag" {
		t.Errorf("keyStore: got %q", r.KeyStore)
	}
	want := map[string]string{"E1": "A2", "E2": "B"}
	if !reflect.DeepEqual(r.Secrets, want) {
		t.Errorf("secrets: got %v, want %v", r.Secrets, want)
	}
}

func TestMappingsRoundTrip(t *testing.T) {
	m := map[string]string{"2": "B", "1": "A", "3": "C", "4": "A"}
	pairs := MappingsFromMap(m)
	if pairs[0] != (Mapping{Name: "A", Env: "1"}) || pairs[1] != (Mapping{Name: "A", Env: "4"}) || pairs[2].Name != "B" || pairs[3].Name != "C" {
		t.Errorf("not sorted by name: %v", pairs)
	}
	if !reflect.DeepEqual(MappingsToMap(pairs), m) {
		t.Errorf("roundtrip mismatch: %v", MappingsToMap(pairs))
	}
}

func TestAggregateStores(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/a.kdbx", Secrets: map[string]string{"GH": "GITHUB_TOKEN", "GH_COPY": "GITHUB_TOKEN"}},
		"install": {Secrets: map[string]string{"NPM": "NPM_TOKEN"}},
		"deploy":  {KeyStore: "/b.kdbx", Secrets: map[string]string{"AWS": "AWS_KEY"}},
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
		"default": {KeyStore: existing, Secrets: map[string]string{"GH": "GITHUB_TOKEN"}},
		"deploy":  {KeyStore: missing, Secrets: map[string]string{"AWS": "AWS_KEY"}},
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
		"default": {KeyStore: "/s", Stdin: []string{"D1"}, Files: []string{"F1"}, Templates: []config.Template{{Name: "T1", Path: "/t1"}}, Askpass: "A1"},
		"docker":  {Stdin: []string{"D2", "D3"}},
	}}

	r := Resolve(cfg, "docker", Overrides{})
	if !reflect.DeepEqual(r.Stdin, []string{"D2", "D3"}) {
		t.Errorf("stdin should be replaced, not merged: got %v", r.Stdin)
	}
	if !reflect.DeepEqual(r.Files, []string{"F1"}) {
		t.Errorf("files: got %v", r.Files)
	}
	if !reflect.DeepEqual(r.Templates, []config.Template{{Name: "T1", Path: "/t1"}}) {
		t.Errorf("templates: got %v", r.Templates)
	}
	if r.Askpass != "A1" {
		t.Errorf("askpass: got %q", r.Askpass)
	}

	r = Resolve(cfg, "docker", Overrides{Stdin: []string{"D4"}, StdinKeepOpen: true, Templates: []config.Template{{Name: "T2", Path: "/t2"}}, Askpass: "A2"})
	if !reflect.DeepEqual(r.Stdin, []string{"D4"}) {
		t.Errorf("stdin override: got %v", r.Stdin)
	}
	if !r.StdinKeepOpen {
		t.Error("stdinKeepOpen override was lost")
	}
	if !reflect.DeepEqual(r.Templates, []config.Template{{Name: "T2", Path: "/t2"}}) {
		t.Errorf("templates override: got %v", r.Templates)
	}
	if r.Askpass != "A2" {
		t.Errorf("askpass override: got %q", r.Askpass)
	}
}

func TestAllTitlesDeduplicates(t *testing.T) {
	r := Resolved{
		Secrets: map[string]string{"ENV": "T", "ENV2": "T"},
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
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "settings.json.tmpl")
	if err := os.WriteFile(tmplPath, []byte(`{"password":"{{T}}"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s"},
		"docker":  {Stdin: []string{"D"}},
		"restic":  {Files: []string{"F"}},
		"admin":   {Templates: []config.Template{{Name: "cfg", Path: tmplPath}}},
		"ssh":     {Askpass: "A"},
	}}
	got := AggregateStores(cfg)["/s"]
	if !reflect.DeepEqual(got, []string{"A", "D", "F", "T"}) {
		t.Errorf("got %v", got)
	}
}

func TestAggregateStoreMappingsChannelTemplate(t *testing.T) {
	dir := t.TempDir()
	tmplPath := filepath.Join(dir, "settings.json.tmpl")
	if err := os.WriteFile(tmplPath, []byte(`{"password":"{{AdminPw}}"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Sections: map[string]config.Section{
		"admin": {KeyStore: "/s", Templates: []config.Template{{Name: "cfg", Path: tmplPath}}},
	}}
	mappings := AggregateStoreMappings(cfg)["/s"]
	if len(mappings) != 1 || mappings[0].Name != "AdminPw" || mappings[0].Env != ChannelTemplate {
		t.Errorf("got %+v", mappings)
	}
}

func TestAggregateStoreMappingsMultipleEnvPerSecret(t *testing.T) {
	cfg := config.Config{Sections: map[string]config.Section{
		"default": {KeyStore: "/s", Secrets: map[string]string{"A_ENV": "T", "B_ENV": "T"}, Stdin: []string{"T", "U"}},
	}}
	got := AggregateStoreMappings(cfg)["/s"]
	want := []Mapping{{Name: "T", Env: "A_ENV"}, {Name: "T", Env: "B_ENV"}, {Name: "U", Env: ChannelStdin}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestValidateEnvNames(t *testing.T) {
	if err := ValidateEnvNames(map[string]string{"GOOD_1": "a", "_ok": "b"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err := ValidateEnvNames(map[string]string{"1BAD": "a", "A=B": "b", "": "c", "OK": "d"})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{`"1BAD"`, `"A=B"`, `""`} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %s", err, name)
		}
	}
	if strings.Contains(err.Error(), `"OK"`) {
		t.Errorf("error %q mentions valid name", err)
	}
}
