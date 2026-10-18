package cmd

import (
	"fmt"
	"slices"
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
	for i := 0; i < len(args); i++ {
		consumed, err := f.parseArg(args, i)
		if err != nil {
			return f, err
		}
		i += consumed
	}
	return f, nil
}

func (f *runFlags) parseArg(args []string, i int) (int, error) {
	arg := args[i]
	if f.setSwitch(arg) {
		return 0, nil
	}
	if name, value, ok := strings.Cut(arg, "="); ok && slices.Contains(valueFlagNames, name) {
		return 0, f.setValue(name, value)
	}

	if !slices.Contains(valueFlagNames, arg) {
		return 0, fmt.Errorf("unknown flag: %s", arg)
	}
	if i+1 >= len(args) {
		return 0, fmt.Errorf("%s requires a value", arg)
	}
	return 1, f.setValue(arg, args[i+1])
}

func (f *runFlags) setSwitch(name string) bool {
	switch name {
	case "--stdin-keep-open":
		f.stdinKeepOpen = true
	case "--dry-run":
		f.dryRun = true
	default:
		return false
	}
	return true
}

func (f *runFlags) setValue(name, value string) error {
	switch name {
	case "--config":
		f.configPath = value
	case "--key-store":
		f.keyStore = value
	case "--secrets":
		return mergeSecretsFlag(f.secrets, value)
	case "--stdin":
		f.stdin = slices.Concat(f.stdin, splitList(value))
	case "--secret-file":
		f.files = slices.Concat(f.files, splitList(value))
	case "--template":
		return f.addTemplates(value)
	case "--askpass":
		f.askpass = value
	case "--askpass-reads":
		return f.setAskpassReads(value)
	}
	return nil
}

func (f *runFlags) addTemplates(spec string) error {
	parsed, err := parseTemplatesFlag(spec)
	if err != nil {
		return err
	}
	f.templates = slices.Concat(f.templates, parsed)
	return nil
}

func (f *runFlags) setAskpassReads(value string) error {
	n, err := parseAskpassReads(value)
	if err != nil {
		return err
	}
	f.askpassReads = n
	return nil
}

func splitNameValue(part string) (string, string, bool) {
	i := strings.LastIndex(part, ":")
	if i < 0 {
		return "", "", false
	}

	name := strings.TrimSpace(part[:i])
	value := strings.TrimSpace(part[i+1:])
	return name, value, name != "" && value != ""
}

func mergeSecretsFlag(dst map[string]string, spec string) error {
	for _, part := range splitList(spec) {
		name, env, ok := splitNameValue(part)
		if !ok {
			return fmt.Errorf("invalid secrets entry %q (expected name:env)", part)
		}
		dst[env] = name
	}
	return nil
}

func parseTemplatesFlag(spec string) ([]config.Template, error) {
	var out []config.Template
	for _, part := range splitList(spec) {
		name, path, ok := splitNameValue(part)
		if !ok {
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
