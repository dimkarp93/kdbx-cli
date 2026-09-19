package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/domain"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/secretpipe"
)

func cmdRun(flags runFlags, child []string) int {
	if len(child) == 0 {
		fmt.Fprintln(os.Stderr, "no command specified after --")
		return 2
	}
	if !flags.dryRun {
		keepass.CheckEngine()
	}

	cfgPath := flags.configPath
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}
	cfg := config.Config{}
	if c, err := config.Load(cfgPath); err == nil {
		cfg = c
	} else if !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	tool := filepath.Base(child[0])
	res := domain.Resolve(cfg, tool, flags.overrides())

	if flags.dryRun {
		printPlan(cfgPath, tool, cfg, res, flags, child)
		return 0
	}

	if res.KeyStore == "" {
		fmt.Fprintln(os.Stderr, "no key-store configured: set it in", cfgPath, "or pass --key-store")
		return 1
	}
	titles := res.AllTitles()
	if len(titles) == 0 {
		fmt.Fprintf(os.Stderr, "no secrets configured for %q in %s (or via flags)\n", tool, cfgPath)
		return 1
	}

	keyStore := config.ExpandHome(res.KeyStore)
	if _, err := os.Stat(keyStore); err != nil {
		fmt.Fprintln(os.Stderr, "key-store not found:", keyStore)
		return 1
	}

	prompt := fmt.Sprintf("Enter password for %s to run %s: ", keyStore, tool)
	out, _, err := domain.UnlockExport(keyStore, keyring.New(cfg.Cache), prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	entries, err := keepass.ParseSecrets(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot parse key-store:", err)
		return 1
	}

	sort.Strings(titles)
	values := make(map[string]string, len(titles))
	var failures []string
	for _, name := range titles {
		val, err := keepass.LookupSecret(entries, name)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		values[name] = val
	}
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "cannot resolve secrets:")
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, "  "+f)
		}
		return 1
	}

	envNames := make([]string, 0, len(res.Secrets))
	for n := range res.Secrets {
		envNames = append(envNames, n)
	}
	sort.Strings(envNames)
	env := os.Environ()
	for _, name := range envNames {
		env = append(env, res.Secrets[name]+"="+values[name])
	}

	stdin := io.Reader(os.Stdin)
	if len(res.Stdin) > 0 {
		payload, err := stdinPayload(res.Stdin, values)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if res.StdinKeepOpen {
			stdin = io.MultiReader(strings.NewReader(payload), os.Stdin)
		} else {
			stdin = strings.NewReader(payload)
		}
	}

	args := child
	var extraFiles []*os.File
	if len(res.Files) > 0 || res.Askpass != "" {
		pipes := secretpipe.NewSet()
		defer pipes.Close()

		paths := make(map[string]string, len(res.Files))
		for _, name := range res.Files {
			f, err := pipes.File(name, values[name])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			defer f.Close()
			paths[name] = fmt.Sprintf("/dev/fd/%d", firstExtraFD+len(extraFiles))
			extraFiles = append(extraFiles, f)
		}
		args, err = substitutePlaceholders(child, paths)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}

		if res.Askpass != "" {
			script, err := pipes.Askpass(res.Askpass, values[res.Askpass])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			env = withAskpass(env, script)
		}
	}

	bin, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "command not found:", args[0])
		return 127
	}
	cmd := exec.Command(bin, args[1:]...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.ExtraFiles = extraFiles
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "command failed:", err)
		return 1
	}
	return 0
}

const firstExtraFD = 3

func stdinPayload(names []string, values map[string]string) (string, error) {
	var b strings.Builder
	for _, name := range names {
		v := values[name]
		if strings.ContainsAny(v, "\r\n") {
			return "", fmt.Errorf("secret %q contains a newline and cannot be sent over stdin", name)
		}
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func placeholder(name string) string { return "{{" + name + "}}" }

func substitutePlaceholders(args []string, paths map[string]string) ([]string, error) {
	out := append([]string{}, args...)
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		ph := placeholder(name)
		found := false
		for i, a := range out {
			if strings.Contains(a, ph) {
				out[i] = strings.ReplaceAll(a, ph, paths[name])
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("placeholder %s for secret %q not found in the command", ph, name)
		}
	}
	return out, nil
}

var askpassEnv = []string{
	"SSH_ASKPASS",
	"SUDO_ASKPASS",
	"GIT_ASKPASS",
	"RESTIC_PASSWORD_COMMAND",
	"BORG_PASSCOMMAND",
}

func withAskpass(env []string, script string) []string {
	set := map[string]string{
		"SSH_ASKPASS_REQUIRE": "force",
		"GIT_TERMINAL_PROMPT": "0",
	}
	for _, k := range askpassEnv {
		set[k] = script
	}
	out := make([]string, 0, len(env)+len(set))
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		if _, override := set[k]; !override {
			out = append(out, kv)
		}
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, k+"="+set[k])
	}
	return out
}
