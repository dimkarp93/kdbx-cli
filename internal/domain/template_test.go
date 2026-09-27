package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func writeTemplate(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTemplateTitlesExtractsUnique(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"user":"{{User}}","password":"{{AdminPw}}","again":"{{User}}"}`)
	got, err := TemplateTitles(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"User", "AdminPw"}) {
		t.Errorf("got %v", got)
	}
}

func TestRenderTemplateRaw(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.env", "PASSWORD={{AdminPw}}\n")
	out, err := RenderTemplate(path, map[string]string{"AdminPw": "s3cret"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "PASSWORD=s3cret\n" {
		t.Errorf("got %q", out)
	}
}

func TestRenderTemplateJSONEscaping(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"user":"Bob","password":"{{AdminPw}}"}`)
	out, err := RenderTemplate(path, map[string]string{"AdminPw": `weird"va\lue` + "\nwith newline"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("rendered file is not valid JSON: %v\n%s", err, out)
	}
	if decoded["password"] != `weird"va\lue`+"\nwith newline" {
		t.Errorf("decoded password mismatch: %q", decoded["password"])
	}
}

func TestRenderTemplateJSONEscapingWithTmplSuffix(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json.tmpl", `{"user":"Bob","password":"{{AdminPw}}"}`)
	out, err := RenderTemplate(path, map[string]string{"AdminPw": `has "quotes" and` + "\nnewline"})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("rendered file is not valid JSON: %v\n%s", err, out)
	}
	if decoded["password"] != `has "quotes" and`+"\nnewline" {
		t.Errorf("decoded password mismatch: %q", decoded["password"])
	}
}

func TestRenderTemplateMissingSecret(t *testing.T) {
	path := writeTemplate(t, t.TempDir(), "settings.json", `{"password":"{{AdminPw}}"}`)
	_, err := RenderTemplate(path, map[string]string{})
	if err == nil {
		t.Fatal("expected an error for a missing secret")
	}
	if !strings.Contains(err.Error(), "AdminPw") {
		t.Errorf("error should name the missing title: %v", err)
	}
}

func TestResolvedTemplateTitlesDedup(t *testing.T) {
	if titles, err := (Resolved{}).TemplateTitles(); err != nil || len(titles) != 0 {
		t.Fatalf("empty resolved should yield no titles, got %v, err=%v", titles, err)
	}

	dir := t.TempDir()
	p1 := writeTemplate(t, dir, "a.json", `{"a":"{{Shared}}"}`)
	p2 := writeTemplate(t, dir, "b.json", `{"b":"{{Shared}}","c":"{{Other}}"}`)
	r := Resolved{Templates: []config.Template{{Name: "t1", Path: p1}, {Name: "t2", Path: p2}}}
	got, err := r.TemplateTitles()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"Shared", "Other"}) {
		t.Errorf("got %v", got)
	}
}
