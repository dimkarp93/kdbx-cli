package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/deliver"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/secretpipe"
	"github.com/dimkarp93/kdbx-cli/internal/store"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

type runPlan struct {
	cfgPath string
	tool    string
	cfg     config.Config
	res     resolve.Resolved
}

func cmdRun(flags runFlags, child []string) int {
	if len(child) == 0 {
		fmt.Fprintln(os.Stderr, "no command specified after --")
		return 2
	}
	if !flags.dryRun {
		keepass.CheckEngine()
	}

	plan, err := resolveRun(flags, child)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if flags.dryRun {
		printPlan(plan, flags, child)
		return 0
	}

	values, err := fetchSecrets(plan)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return runChild(plan.res, values, child)
}

func resolveRun(flags runFlags, child []string) (runPlan, error) {
	cfgPath, cfg, err := loadRunConfig(flags.configPath)
	if err != nil {
		return runPlan{}, err
	}

	tool := filepath.Base(child[0])
	res := resolve.Resolve(cfg, tool, flags.overrides())
	if err := resolve.ValidateEnvNames(res.Secrets); err != nil {
		return runPlan{}, err
	}
	return runPlan{cfgPath: cfgPath, tool: tool, cfg: cfg, res: res}, nil
}

func fetchSecrets(plan runPlan) (map[string]string, error) {
	titles, err := runTitles(plan.cfgPath, plan.tool, plan.res)
	if err != nil {
		return nil, err
	}
	keyStore, err := existingKeyStore(plan.res.KeyStore)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf("Enter password for %s to run %s: ", keyStore, plan.tool)
	cache := keyring.New(plan.cfg.Cache)
	granted := utils.ToSet(slices.Concat(resolve.ConfiguredTitles(plan.cfg, plan.tool), cache.Granted(keyStore)))
	out, _, err := store.UnlockExport(keyStore, cache, prompt, titlesGranted(titles, granted))
	if err != nil {
		return nil, err
	}

	return lookupRunSecrets(out, titles)
}

func runTitles(cfgPath, tool string, res resolve.Resolved) ([]string, error) {
	if res.KeyStore == "" {
		return nil, errNoKeyStore(cfgPath)
	}

	titles, err := requestedTitles(res)
	if err != nil {
		return nil, err
	}
	if len(titles) == 0 {
		return nil, fmt.Errorf("no secrets configured for %q in %s (or via flags)", tool, cfgPath)
	}
	return titles, nil
}

func lookupRunSecrets(export string, titles []string) (map[string]string, error) {
	entries, err := parseEntries(export)
	if err != nil {
		return nil, err
	}

	values, failures := lookupAll(entries, slices.Sorted(slices.Values(titles)))
	if len(failures) > 0 {
		return nil, failuresError("cannot resolve secrets:", failures)
	}
	return values, nil
}

type childSpec struct {
	args       []string
	env        []string
	stdin      io.Reader
	extraFiles []*os.File
}

func runChild(res resolve.Resolved, values map[string]string, child []string) int {
	pipes := secretpipe.NewSet()
	defer pipes.Close()

	spec, err := prepareChild(pipes, res, values, child)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer deliver.CloseFiles(spec.extraFiles)

	return execChild(spec)
}

func prepareChild(pipes *secretpipe.Set, res resolve.Resolved, values map[string]string, child []string) (childSpec, error) {
	stdin, err := deliver.Stdin(res.Stdin, res.StdinKeepOpen, values, os.Stdin)
	if err != nil {
		return childSpec{}, err
	}

	files, err := deliver.RenderFiles(res.Files, res.Templates, values)
	if err != nil {
		return childSpec{}, err
	}
	args, err := deliver.SubstitutePlaceholders(child, files.Paths())
	if err != nil {
		return childSpec{}, err
	}

	env, err := childEnv(pipes, res, values)
	if err != nil {
		return childSpec{}, err
	}

	extraFiles, err := files.Open(pipes)
	if err != nil {
		return childSpec{}, err
	}
	return childSpec{args: args, env: env, stdin: stdin, extraFiles: extraFiles}, nil
}

func childEnv(pipes *secretpipe.Set, res resolve.Resolved, values map[string]string) ([]string, error) {
	env := slices.Concat(deliver.InheritedEnv(os.Environ()), deliver.SecretEnv(res.Secrets, values))
	if res.Askpass == "" {
		return env, nil
	}

	script, err := pipes.Askpass(res.Askpass, values[res.Askpass], res.AskpassReads)
	if err != nil {
		return nil, err
	}
	return deliver.WithAskpass(env, script), nil
}

func execChild(spec childSpec) int {
	bin, err := exec.LookPath(spec.args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "command not found:", spec.args[0])
		return 127
	}

	cmd := exec.Command(bin, spec.args[1:]...)
	cmd.Env = spec.env
	cmd.Stdin = spec.stdin
	cmd.ExtraFiles = spec.extraFiles
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return exitCode(cmd.Run())
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	fmt.Fprintln(os.Stderr, "command failed:", err)
	return 1
}

const firstExtraFD = 3

const passwordEnv = "KDBX_CLI_PASSWORD"

var askpassEnv = []string{
	"SSH_ASKPASS",
	"SUDO_ASKPASS",
	"GIT_ASKPASS",
	"RESTIC_PASSWORD_COMMAND",
	"BORG_PASSCOMMAND",
}
