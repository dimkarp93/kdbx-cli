# kdbx-cli

`kdbx-cli` is a wrapper for CLI tools that takes secrets from an encrypted `.kdbx` store and hands them to a child command: through environment variables, its stdin, a file or an askpass helper.

Why: many tools require tokens to be passed via env variables. Doing that by hand from the shell is inconvenient and unsafe — the secret ends up in the command history and in plaintext files. `kdbx-cli` solves this: the secret value is taken from an encrypted `.kdbx`, and the invocation looks like this:

```sh
kdbx-cli --config ~/.config/kdbx-cli/install_secrets -- install user/repo
```

There is no secret on the command line — the history stays clean. The secret exists only in the environment of the child process.

This is the same pattern as `op run -- cmd` (1Password CLI), `envchain`, `aws-vault exec`, `sops exec-env` — but self-hosted, on top of `keepassxc`.

## Requirements

`keepassxc-cli` must be installed (it ships with KeePassXC):

```sh
# Debian/Ubuntu
sudo apt install keepassxc
# Arch Linux
sudo pacman -S keepassxc
# macOS
brew install keepassxc
```

## Installation

Via the [`dimkarp93/install`](https://github.com/dimkarp93/install) installer:

```sh
github_install.sh dimkarp93/kdbx-cli
# or as a one-liner:
curl -fsSL https://raw.githubusercontent.com/dimkarp93/install/master/install.sh | sh -s -- dimkarp93/kdbx-cli
```

## Usage

```
kdbx-cli [--config <path>] [--key-store <path>] [--secrets=name:env,...]
         [--stdin=name,...] [--stdin-keep-open] [--secret-file=name,...]
         [--askpass=name] [--dry-run] -- <cmd> [args...]
kdbx-cli config [--config <path>]
kdbx-cli version | --version | -v
```

Everything after `--` is the command to run. The config section is selected by the **base name** of the command's first word (`install` in the example; for `/usr/bin/sudo` it is `sudo`); if there is no such section, `default` is used. Flags must come **before** `--`.

Flags:

- `--config <path>` — path to the config. Defaults to `~/.config/kdbx-cli/default`.
- `--key-store <path>` — full path to the `.kdbx` file. Overrides the value from the config.
- `--secrets=name:env,...` — mapping of secrets to env variables. Merged on top of the config.
- `--stdin=name,...` — write the secrets to the command's stdin, one per line in the given order, then close it.
- `--stdin-keep-open` — keep stdin open after the secrets and pass the rest of our own stdin through.
- `--secret-file=name,...` — expose the secret as a file and substitute its path for the `{{name}}` placeholder in the command.
- `--askpass=name` — serve the secret through an askpass helper (for commands that only read from the terminal).
- `--dry-run` — do not run the command and do not touch the store: print the resolved plan (see below).

On startup `kdbx-cli` asks for the store password (input is hidden, read from `/dev/tty`). Cancel with `Ctrl+C` / `Ctrl+D`.

### Example

```sh
kdbx-cli --key-store ~/secrets/tokens.kdbx --secrets=GITHUB_TOKEN:GH_TOKEN -- gh repo list
```

`kdbx-cli` reads the entry titled `GITHUB_TOKEN` from `tokens.kdbx`, puts its password into the `GH_TOKEN` variable and runs `gh repo list`.

### Preview: `--dry-run`

With the `--dry-run` flag the command is not executed, no password is requested and the `.kdbx` is not read. The resolved plan is printed instead: which config is used, which section is applied, the active mappings and the resulting command (secret values are replaced with the `<secret from Title>` placeholder):

```sh
$ kdbx-cli --dry-run -- install user/repo
Dry run — the command will NOT be executed.

Config file:  /home/user/.config/kdbx-cli/default
Tool:         install
Section:      "install" (merged over "default")
Key-store:    ~/.config/kdbx-cli/store.kdbx

Mappings (env ← secret):
  GH_TOKEN  ← GITHUB_TOKEN
  NPM_TOKEN ← NPM_TOKEN

Command:
  GH_TOKEN=<secret from GITHUB_TOKEN> NPM_TOKEN=<secret from NPM_TOKEN> install user/repo
```

## Config

The config is a JSON object with the fields `sections` (a set of sections named after tools plus `default`) and an optional `cached` (see [Password caching](#password-caching)). Every section has:

- `key-store` — full path to the `.kdbx` file;
- `secrets` — mapping `store_entry_name: env_name`;
- `stdin` — the list of secrets written to the command's stdin (the `--stdin` counterpart);
- `stdin-keep-open` — `true` to keep stdin open after the secrets (the `--stdin-keep-open` counterpart);
- `files` — the list of secrets handed over as files through the `{{Title}}` placeholder (the `--secret-file` counterpart);
- `askpass` — a single secret served through the askpass helper (the `--askpass` counterpart).

Every delivery channel can be set up both from a flag and from the config — the flags merely override the config, there is no CLI-only channel. See [Delivery channels](#delivery-channels) for the details.

```json
{
  "sections": {
    "default": {
      "key-store": "~/.config/kdbx-cli/store.kdbx",
      "secrets": { "GITHUB_TOKEN": "GH_TOKEN" }
    },
    "install": {
      "secrets": { "NPM_TOKEN": "NPM_TOKEN" }
    }
  },
  "cached": { "enabled": true, "ttl": "10m" }
}
```

**Merging:** `default` is the base. The tool section is layered on top: it overrides `key-store` (if set) and adds/overrides secret mappings. For the example above, the `install` command gets the shared `key-store` from `default` and both secrets — `GITHUB_TOKEN` and `NPM_TOKEN`. The `--key-store` / `--secrets` flags override the result.

### How a secret is looked up

The "secret name" in the config is the **Title** of an entry in the `.kdbx`, and the value substituted is the **Password** field of that entry. If the store has several entries with the same Title, specify the full path `Group/Subgroup/Title`:

```json
"secrets": { "web/API_KEY": "API_KEY" }
```

## Delivery channels

Not every tool reads its secret from the environment. `kdbx-cli` supports four channels; they can be combined in one section.

### `secrets` — environment variables

The default channel, described above.

### `stdin` — writing to the command's stdin

The secrets are written to the child's stdin, one line per secret in the given order, after which stdin is closed (which is exactly what `--password-stdin` flags expect). It is a list, not a single value: some tools ask for the password twice.

```sh
kdbx-cli --stdin=ghcr-token -- docker login ghcr.io -u me --password-stdin
kdbx-cli --stdin=gh-token   -- gh auth login --with-token
kdbx-cli --stdin=vault-root -- vault login -
kdbx-cli --stdin=new-pw,new-pw -- keepassxc-cli db-create -p ~/new.kdbx
```

`--stdin-keep-open` is for commands that read the password from stdin and then expect data there as well:

```sh
kdbx-cli --stdin=sudo-pw --stdin-keep-open -- sudo -S tee /etc/foo.conf < local.conf
```

A secret containing a newline cannot be delivered this way — `kdbx-cli` exits with an error.

### `files` — the secret as a file

The secret is placed into an anonymous in-memory file (`memfd`), the child receives it as `/dev/fd/N`, and `kdbx-cli` substitutes that path for the `{{Title}}` placeholder in the arguments. The secret never gets a name on any filesystem. The file can be read any number of times.

```sh
kdbx-cli --secret-file=restic-repo -- restic -r sftp:backup:/b --password-file '{{restic-repo}}' snapshots
kdbx-cli --secret-file=mysql-ini   -- mysql --defaults-extra-file='{{mysql-ini}}' mydb
kdbx-cli --secret-file=gpg-pass    -- gpg --batch --pinentry-mode loopback --passphrase-file '{{gpg-pass}}' -d file.gpg
```

Put the placeholder in single quotes so the shell leaves it alone. If the placeholder is absent from the command, `kdbx-cli` exits with an error instead of silently running it.

### `askpass` — for commands that only read from the terminal

`ssh`, `sudo` and `git` open `/dev/tty` directly when asking for a password, so writing to stdin does not help. The standard way around it is an askpass helper. `kdbx-cli` creates one and sets `SSH_ASKPASS`, `SSH_ASKPASS_REQUIRE=force`, `SUDO_ASKPASS`, `GIT_ASKPASS`, `GIT_TERMINAL_PROMPT=0`, `RESTIC_PASSWORD_COMMAND` and `BORG_PASSCOMMAND`.

```sh
kdbx-cli --askpass=ssh-key-pass -- ssh -T git@github.com
kdbx-cli --askpass=gitlab-pat   -- git push origin master
kdbx-cli --askpass=sudo-pw      -- sudo -A apt update
kdbx-cli --askpass=restic-pw    -- restic -r sftp:backup:/b snapshots
```

`sudo` only looks at `SUDO_ASKPASS` with the `-A` flag — you have to pass it yourself.

### In the config

```json
{
  "sections": {
    "default": { "key-store": "~/.config/kdbx-cli/store.kdbx" },
    "docker":  { "stdin": ["ghcr-token"] },
    "sudo":    { "stdin": ["sudo-pw"], "stdin-keep-open": true },
    "restic":  { "files": ["restic-repo"] },
    "ssh":     { "askpass": "ssh-key-pass" }
  }
}
```

Channels combine within one section — `restic`, for example, can take `B2_ACCOUNT_KEY` from the environment, the repository password from a file and the ssh key passphrase through askpass:

```json
"restic": {
  "secrets": { "B2_KEY": "B2_ACCOUNT_KEY" },
  "files":   ["restic-pw"],
  "askpass": "ssh-key-pass"
}
```

#### Precedence

The layers are applied in the order `default` → tool section → flags, by these rules:

| Field | How it is applied |
| --- | --- |
| `key-store` | replaced when set to a non-empty value |
| `secrets` | extended entry by entry (on a key collision the later layer wins) |
| `stdin`, `files` | replaced as a whole: the order of the lines matters, and concatenating the lists would give a surprising result |
| `stdin-keep-open` | enabled when `true` on at least one layer |
| `askpass` | replaced when set to a non-empty value |

Hence a few consequences that are easy to miss:

- `--stdin=a --stdin=b` in one invocation accumulate into `a,b`, but together they replace the whole `stdin` list from the config instead of extending it. The same goes for `--secret-file`.
- A `stdin-keep-open` or `askpass` set in `default` cannot be turned off from a tool section or by a flag — there are no `--no-stdin-keep-open` / `--no-askpass` flags. Keep such settings in the section of the specific tool rather than in `default`.
- `--secrets` does not disable the mappings from the config; it only adds its own and overrides those with the same name.

To check the result without touching the store or typing a password, use `--dry-run` — it prints the `Stdin`, `Files` and `Askpass` blocks along with the section that was applied:

```sh
$ kdbx-cli --dry-run -- restic -r sftp:b:/b --password-file '{{restic-pw}}' snapshots
...
Section:      "restic" (merged over "default")

Mappings (env ← secret):
  B2_ACCOUNT_KEY ← B2_KEY

Files (placeholder ← secret):
  {{restic-pw}} ← restic-pw

Askpass:      ssh-key-pass
              SSH_ASKPASS, SUDO_ASKPASS, GIT_ASKPASS, RESTIC_PASSWORD_COMMAND, BORG_PASSCOMMAND

Command:
  B2_ACCOUNT_KEY=<secret from B2_KEY> restic -r sftp:b:/b --password-file '<file with restic-pw>' snapshots
```

`kdbx-cli check` and `kdbx-cli show` cover the secrets of all four channels.

## The `config` command

`kdbx-cli config` interactively configures the `default` section (the `key-store` path and the secret mappings) and writes the config:

```sh
kdbx-cli config
# or into an arbitrary file:
kdbx-cli config --config ~/.config/kdbx-cli/install_secrets
```

A TUI opens in the terminal:

- if the config already exists, the fields are **prefilled** with the current values;
- the `key-store` field supports **path completion**: `Tab` — complete (and expand `~` into the full path), `↑/↓` — cycle through the options in the current directory, `Alt+Backspace` — delete the last path segment (up to `/`);
- secret mappings are shown **line by line** and edited with hotkeys: `a` — add, `e` — edit, `d` — delete, `↑/↓` — select, `Tab` — switch between the path field and the list, `Ctrl+S` — save, `Esc` — cancel. The hotkey legend is always visible at the bottom.

If stdin/stdout is not a terminal (a pipe, a script), `config` falls back to simple line-by-line input without the TUI.

The TUI edits only the `default` section, and only its `key-store`, the `secrets` mappings and the password cache. The `stdin`, `files` and `askpass` channels and the sections of other tools are not configurable there — edit them directly in the JSON config file; saving from the TUI keeps them as they are and overwrites nothing.

After saving, `config` checks the store of the `default` section:

- if the `.kdbx` file does not exist, it offers to create it (asking for a new password);
- then it verifies that every specified `Title` is present in the store; missing ones are listed and it offers to add them as empty entries.

With the `-y` flag everything missing (the file, the secrets) is created without questions.

## The `check` command

`kdbx-cli check` verifies that all `.kdbx` files referenced by the config (across all sections, taking merging with `default` into account) contain entries with every required `Title`. Missing ones are printed **grouped by file**, after which it offers to add them as empty entries.

```sh
kdbx-cli check
kdbx-cli check --config ~/.config/kdbx-cli/install_secrets
kdbx-cli check -y          # add all missing entries without confirmation
```

Sample output:

```
Missing secrets:
  /home/user/.config/kdbx-cli/store.kdbx
    - NPM_TOKEN
  /home/user/work/deploy.kdbx  (key-store does not exist)
    - AWS_KEY
```

A password is requested for every `.kdbx` (it is needed both to read and to add entries).

## The `show` command

`kdbx-cli show` displays the current config: the file path and the list of `.kdbx` stores with their mappings (`name → env`). You can navigate the stores with the `↑/↓` arrows, and `Enter` opens the selected file in the **KeePassXC GUI** (the `keepassxc` binary).

```sh
kdbx-cli show
kdbx-cli show --config ~/.config/kdbx-cli/install_secrets
```

- Files missing on disk are marked `(missing)`; they cannot be opened.
- `q` / `Esc` — quit.
- If stdin/stdout is not a terminal, `show` simply prints the config without navigation.

## Password caching

Within a single `kdbx-cli` invocation the database password is asked once. To avoid retyping it **between** invocations during a session, caching can be enabled in the config:

```json
"cached": { "enabled": true, "ttl": "10m" }
```

- `enabled` — enable the cache (disabled by default).
- `ttl` — entry lifetime (Go `time.ParseDuration` format: `30s`, `10m`, `2h`; defaults to `10m`).

The password is stored in the **OS keyring** via Secret Service (gnome-keyring / KWallet) and is controlled only through the config (there are no CLI flags or env variables). To drop the cache: `kdbx-cli forget` (clears the entries for every `.kdbx` from the config).

**What this implies and the risks:**

- What is cached is the **master password** of the `.kdbx` — it opens the **whole** database, not just the injected variables. That is a more valuable target than the environment of a child process.
- In the keyring the password is encrypted on disk and decrypted in the daemon's memory for the duration of the login session; you can inspect/revoke it in **seahorse** ("Passwords and Keys").
- The basic Secret Service has **no** per-application separation: any process running as the same user can read the entry (the same trust boundary as `/proc/<pid>/environ`).
- If Secret Service is unavailable (headless, SSH, no session bus), the cache simply does not work and the password is requested as usual (the command does not fail).

## Security limitations

The secret never reaches the command line (`/proc/<pid>/cmdline` is readable by any process) and is never written to disk in plaintext: the file channel uses an anonymous in-memory file, and askpass a FIFO in a `0700` directory.

In the `secrets` channel, though, `kdbx-cli` injects secrets into the environment of a child process, and a process's env variables are readable via `/proc/<pid>/environ` by the same user (and root). This is the common tradeoff of this whole class of tools (`op run`, `envchain`, `aws-vault`) — but it is radically safer than keeping secrets in the shell history or in plaintext files. If you need protection from neighbouring processes of the same user reading the environment, this approach (like its analogues) is not suitable.

## Development

```sh
make build       # build ./kdbx-cli
make unit-test   # unit tests
make e2e-test    # e2e (requires keepassxc-cli)
make test        # everything at once
```
