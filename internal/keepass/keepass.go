package keepass

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

type kpFile struct {
	XMLName xml.Name `xml:"KeePassFile"`
	Root    kpRoot   `xml:"Root"`
}

type kpRoot struct {
	Group kpGroup `xml:"Group"`
}

type kpGroup struct {
	Name    string       `xml:"Name"`
	Groups  []kpGroup    `xml:"Group"`
	Entries []kpXMLEntry `xml:"Entry"`
}

type kpXMLEntry struct {
	Strings []kpString `xml:"String"`
}

type kpString struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type Entry struct {
	Path  string
	Title string
	Value string
}

func CheckEngine() string {
	path, err := exec.LookPath("keepassxc-cli")
	if err != nil {
		fmt.Fprintln(os.Stderr, "keepassxc-cli not found in PATH.")
		fmt.Fprintln(os.Stderr, "Install it:")
		fmt.Fprintln(os.Stderr, "  Debian/Ubuntu:  sudo apt install keepassxc")
		fmt.Fprintln(os.Stderr, "  Arch Linux:     sudo pacman -S keepassxc")
		fmt.Fprintln(os.Stderr, "  macOS:          brew install keepassxc")
		os.Exit(1)
	}
	if err := checkEngineFile(path); err != nil {
		fmt.Fprintf(os.Stderr, "refusing to run %s: %v\n", path, err)
		os.Exit(1)
	}
	enginePath = path
	return path
}

var enginePath string

func checkEngineFile(path string) error {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	for _, p := range []string{filepath.Dir(path), real, filepath.Dir(real)} {
		fi, err := os.Stat(p)
		if err != nil {
			return err
		}
		if fi.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("%s is writable by group or others", p)
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid != 0 && int(st.Uid) != os.Getuid() {
			return fmt.Errorf("%s is owned by uid %d", p, st.Uid)
		}
	}
	return nil
}

func Run(stdinData string, args ...string) (string, error) {
	cmd := exec.Command(engine(), args...)
	cmd.Stdin = strings.NewReader(stdinData)
	out, err := cmd.Output()
	return string(out), err
}

func engine() string {
	if enginePath != "" {
		return enginePath
	}
	return CheckEngine()
}

func ParseSecrets(xmlData string) ([]Entry, error) {
	var kpf kpFile
	if err := xml.Unmarshal([]byte(xmlData), &kpf); err != nil {
		return nil, err
	}
	return groupEntries(kpf.Root.Group, nil, 0), nil
}

func groupEntries(g kpGroup, prefix []string, depth int) []Entry {
	path := groupPath(g, prefix, depth)
	own := utils.Filter(utils.Map(g.Entries, func(e kpXMLEntry) Entry { return toEntry(e, path) }), hasTitle)
	nested := utils.FlatMap(g.Groups, func(child kpGroup) []Entry { return groupEntries(child, path, depth+1) })
	return slices.Concat(own, nested)
}

func groupPath(g kpGroup, prefix []string, depth int) []string {
	if depth == 0 {
		return prefix
	}
	return slices.Concat(prefix, []string{g.Name})
}

func toEntry(e kpXMLEntry, groups []string) Entry {
	var title, password string
	for _, s := range e.Strings {
		switch s.Key {
		case "Title":
			title = s.Value
		case "Password":
			password = s.Value
		}
	}
	return Entry{Path: strings.Join(slices.Concat(groups, []string{title}), "/"), Title: title, Value: password}
}

func hasTitle(e Entry) bool {
	return e.Title != ""
}

func LookupSecret(entries []Entry, name string) (string, error) {
	byPath := strings.Contains(name, "/")
	matches := utils.Filter(entries, func(e Entry) bool {
		return (byPath && e.Path == name) || (!byPath && e.Title == name)
	})

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("secret %q not found in key-store", name)
	case 1:
		return matches[0].Value, nil
	default:
		paths := strings.Join(utils.Map(matches, entryPath), ", ")
		return "", fmt.Errorf("secret %q is ambiguous, matches: %s (use full Group/Title path)", name, paths)
	}
}

func entryPath(e Entry) string {
	return e.Path
}
