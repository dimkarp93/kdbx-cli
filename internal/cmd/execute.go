package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dimkarp93/install-libs/buildinfo"
)

func Execute(info buildinfo.Info) {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	if info.Handle(args) {
		return
	}

	switch args[0] {
	case "help", "--help", "-h":
		usage()
		return
	case "config":
		path, yes := parseConfigArgs("config", args[1:])
		cmdConfig(path, yes)
		return
	case "check":
		path, yes := parseConfigArgs("check", args[1:])
		cmdCheck(path, yes)
		return
	case "show":
		path, _ := parseConfigArgs("show", args[1:])
		cmdShow(path)
		return
	case "forget":
		path, _ := parseConfigArgs("forget", args[1:])
		cmdForget(path)
		return
	}

	left, child, hasSep := splitArgs(args)

	if !hasSep {
		fmt.Fprintln(os.Stderr, "missing -- separator before the command to run")
		usage()
		os.Exit(2)
	}

	flags, err := parseRunFlags(left)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	os.Exit(cmdRun(flags, child))
}

func parseConfigArgs(cmd string, args []string) (string, bool) {
	path := ""
	assumeYes := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config":
			if i+1 < len(args) {
				path = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--config="):
			path = strings.TrimPrefix(a, "--config=")
		case a == "-y" || a == "--yes":
			assumeYes = true
		default:
			fmt.Fprintf(os.Stderr, "unknown argument for %s: %s\n", cmd, a)
			os.Exit(2)
		}
	}
	return path, assumeYes
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  kdbx-cli [--config <path>] [--key-store <path>] [--secrets=name:env,...]
           [--stdin=name,...] [--stdin-keep-open] [--secret-file=name,...]
           [--askpass=name] [--dry-run] -- <cmd> [args...]
  kdbx-cli config [--config <path>] [-y]
  kdbx-cli check  [--config <path>] [-y]
  kdbx-cli show   [--config <path>]
  kdbx-cli forget [--config <path>]
  kdbx-cli version | --version | -v | --origin | --buildinfo

Runs <cmd> with secrets from a .kdbx key-store delivered through environment
variables, its stdin, a file or an askpass helper. Secrets never touch your
shell history, the command line or the disk.

Commands:
  config   Edit the default section interactively; then verify/create the
           key-store and its secrets.
  check    Verify every .kdbx in the config has all referenced secrets;
           offer to add the missing ones as empty entries.
  show     Display the config; navigate key-stores with ↑/↓ and press Enter
           to open the selected .kdbx in the KeePassXC GUI.
  forget   Clear cached key-store passwords (from the OS keyring) for every
           .kdbx referenced by the config.

Password caching is opt-in via the config's "cached" section
({"enabled": true, "ttl": "10m"}); it stores the .kdbx master password in the
OS keyring (Secret Service). Disabled by default.

Flags:
  --config <path>          Config file (default: ~/.config/kdbx-cli/default)
  --key-store <path>       Path to the .kdbx file (overrides config)
  --secrets=name:env,...   Secret-to-env mapping (merged over config)
  --stdin=name,...         Write these secrets to the command's stdin, one per
                           line in the given order, then close it
                           (docker login --password-stdin, gh auth login
                           --with-token, vault login -)
  --stdin-keep-open        Keep stdin open after the secrets and pass the rest
                           of our own stdin through (sudo -S)
  --secret-file=name,...   Expose each secret as a file and substitute its path
                           for the {{name}} placeholder in <cmd>
                           (restic --password-file {{name}})
  --askpass=name           Serve this secret through an askpass helper for
                           commands that only read from the terminal; sets
                           SSH_ASKPASS, SUDO_ASKPASS, GIT_ASKPASS,
                           RESTIC_PASSWORD_COMMAND, BORG_PASSCOMMAND
                           (sudo still needs its own -A flag)
  --dry-run                Print the resolved plan and exit; do not read the
                           key-store, prompt for a password, or run the command
  -y, --yes                For config/check: assume yes (create missing
                           key-stores and empty secrets without prompting)

The first word of <cmd> selects the config section (falling back to "default").
Requires keepassxc-cli in PATH.
`)
}
