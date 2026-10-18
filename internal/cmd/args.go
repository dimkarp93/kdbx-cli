package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

type runFlags struct {
	configPath    string
	keyStore      string
	secrets       map[string]string
	stdin         []string
	stdinKeepOpen bool
	files         []string
	templates     []config.Template
	askpass       string
	askpassReads  int
	dryRun        bool
}

var valueFlagNames = []string{
	"--config", "--key-store", "--secrets", "--stdin", "--secret-file", "--template", "--askpass", "--askpass-reads",
}

func splitArgs(args []string) (left, child []string, hasSep bool) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:], true
		}
	}
	return args, nil, false
}

func parseRunFlags(args []string) (runFlags, error) {
	f := runFlags{secrets: map[string]string{}}
	needValue := func(i int) (string, error) {
		if i+1 >= len(args) {
			return "", fmt.Errorf("%s requires a value", args[i])
		}
		return args[i+1], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.configPath = v
			i++
		case strings.HasPrefix(a, "--config="):
			f.configPath = strings.TrimPrefix(a, "--config=")
		case a == "--key-store":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.keyStore = v
			i++
		case strings.HasPrefix(a, "--key-store="):
			f.keyStore = strings.TrimPrefix(a, "--key-store=")
		case a == "--secrets":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			if err := mergeSecretsFlag(f.secrets, v); err != nil {
				return f, err
			}
			i++
		case strings.HasPrefix(a, "--secrets="):
			if err := mergeSecretsFlag(f.secrets, strings.TrimPrefix(a, "--secrets=")); err != nil {
				return f, err
			}
		case a == "--stdin":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.stdin = append(f.stdin, splitList(v)...)
			i++
		case strings.HasPrefix(a, "--stdin="):
			f.stdin = append(f.stdin, splitList(strings.TrimPrefix(a, "--stdin="))...)
		case a == "--stdin-keep-open":
			f.stdinKeepOpen = true
		case a == "--secret-file":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.files = append(f.files, splitList(v)...)
			i++
		case strings.HasPrefix(a, "--secret-file="):
			f.files = append(f.files, splitList(strings.TrimPrefix(a, "--secret-file="))...)
		case a == "--template":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			parsed, err := parseTemplatesFlag(v)
			if err != nil {
				return f, err
			}
			f.templates = append(f.templates, parsed...)
			i++
		case strings.HasPrefix(a, "--template="):
			parsed, err := parseTemplatesFlag(strings.TrimPrefix(a, "--template="))
			if err != nil {
				return f, err
			}
			f.templates = append(f.templates, parsed...)
		case a == "--askpass":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			f.askpass = v
			i++
		case strings.HasPrefix(a, "--askpass="):
			f.askpass = strings.TrimPrefix(a, "--askpass=")
		case a == "--askpass-reads":
			v, err := needValue(i)
			if err != nil {
				return f, err
			}
			if f.askpassReads, err = parseAskpassReads(v); err != nil {
				return f, err
			}
			i++
		case strings.HasPrefix(a, "--askpass-reads="):
			var err error
			if f.askpassReads, err = parseAskpassReads(strings.TrimPrefix(a, "--askpass-reads=")); err != nil {
				return f, err
			}
		case a == "--dry-run":
			f.dryRun = true
		default:
			return f, fmt.Errorf("unknown flag: %s", a)
		}
	}
	return f, nil
}

func mergeSecretsFlag(dst map[string]string, spec string) error {
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.LastIndex(part, ":")
		if i < 0 {
			return fmt.Errorf("invalid secrets entry %q (expected name:env)", part)
		}
		name := strings.TrimSpace(part[:i])
		env := strings.TrimSpace(part[i+1:])
		if name == "" || env == "" {
			return fmt.Errorf("invalid secrets entry %q (expected name:env)", part)
		}
		dst[env] = name
	}
	return nil
}

func parseTemplatesFlag(spec string) ([]config.Template, error) {
	var out []config.Template
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		i := strings.LastIndex(part, ":")
		if i < 0 {
			return nil, fmt.Errorf("invalid template entry %q (expected name:path)", part)
		}
		name := strings.TrimSpace(part[:i])
		path := strings.TrimSpace(part[i+1:])
		if name == "" || path == "" {
			return nil, fmt.Errorf("invalid template entry %q (expected name:path)", part)
		}
		out = append(out, config.Template{Name: name, Path: path})
	}
	return out, nil
}

func parseAskpassReads(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid --askpass-reads %q (expected a positive number)", v)
	}
	return n, nil
}

func splitList(spec string) []string {
	var out []string
	for _, part := range strings.Split(spec, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (f runFlags) overrides() resolve.Overrides {
	return resolve.Overrides{
		KeyStore:      f.keyStore,
		Secrets:       f.secrets,
		Stdin:         f.stdin,
		StdinKeepOpen: f.stdinKeepOpen,
		Files:         f.files,
		Templates:     f.templates,
		Askpass:       f.askpass,
		AskpassReads:  f.askpassReads,
	}
}
