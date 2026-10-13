package keepass

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	bin := enginePath
	if bin == "" {
		bin = CheckEngine()
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(stdinData)
	out, err := cmd.Output()
	return string(out), err
}

func ParseSecrets(xmlData string) ([]Entry, error) {
	var kpf kpFile
	if err := xml.Unmarshal([]byte(xmlData), &kpf); err != nil {
		return nil, err
	}

	var out []Entry
	var walk func(g kpGroup, prefix []string, depth int)
	walk = func(g kpGroup, prefix []string, depth int) {
		cur := prefix
		if depth > 0 {
			cur = append(append([]string{}, prefix...), g.Name)
		}
		for _, e := range g.Entries {
			var title, password string
			for _, s := range e.Strings {
				switch s.Key {
				case "Title":
					title = s.Value
				case "Password":
					password = s.Value
				}
			}
			if title == "" {
				continue
			}
			path := title
			if len(cur) > 0 {
				path = strings.Join(append(append([]string{}, cur...), title), "/")
			}
			out = append(out, Entry{Path: path, Title: title, Value: password})
		}
		for _, child := range g.Groups {
			walk(child, cur, depth+1)
		}
	}
	walk(kpf.Root.Group, nil, 0)
	return out, nil
}

func LookupSecret(entries []Entry, name string) (string, error) {
	byPath := strings.Contains(name, "/")
	var matches []Entry
	for _, e := range entries {
		if (byPath && e.Path == name) || (!byPath && e.Title == name) {
			matches = append(matches, e)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("secret %q not found in key-store", name)
	case 1:
		return matches[0].Value, nil
	default:
		var paths []string
		for _, m := range matches {
			paths = append(paths, m.Path)
		}
		return "", fmt.Errorf("secret %q is ambiguous, matches: %s (use full Group/Title path)", name, strings.Join(paths, ", "))
	}
}
