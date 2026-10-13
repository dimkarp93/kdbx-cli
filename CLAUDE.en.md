# CLAUDE.md

## Code style

**Do not write comments in the code.** No explanatory comments on functions, no section headers (`// --- Foo ---`), no parameter descriptions and so on. The code must be self-documenting through identifier names.

The only exception is when without a comment the reader would not understand *why* (a hidden invariant, a bug workaround, a non-obvious API limitation). In such rare cases — one short line.

## Language

All code, scripts, CLI messages and comments in CI configs are **English only**.

Documentation is maintained in two languages, as file pairs:

- `<name>.ru.md` — the Russian version (the source of truth);
- `<name>.en.md` — the English version;
- `<name>.md` — a symlink to `<name>.en.md` (English is the default).

Changing a document requires editing **both** language versions. Only command identifiers, flags, config keys and shell examples are kept in their original form.

## Purpose

`kdbx-cli` is a wrapper that runs a command with secrets from a `.kdbx` store. There are five delivery channels: environment variables (`secrets`), the command's stdin (`stdin`), a file via `memfd` + `/dev/fd/N` (`files`), a file rendered from a template with secrets interpolated into its content, delivered the same way (`templates`), and an askpass helper (`askpass`). Secrets do not end up in the shell history, in argv, or on disk in plaintext. Reading the `.kdbx` goes through the external `keepassxc-cli` (there is no in-house crypto).

```
kdbx-cli [--config <path>] [--key-store <path>] [--secrets=name:env,...]
         [--stdin=name,...] [--stdin-keep-open] [--secret-file=name,...]
         [--template=name:path,...] [--askpass=name] [--askpass-reads=N]
         [--dry-run] -- <cmd> [args...]
kdbx-cli config  [--config <path>] [-y]
kdbx-cli check   [--config <path>] [-y]
kdbx-cli show    [--config <path>]
kdbx-cli forget  [--config <path>]
kdbx-cli migrate [--config <path>] [--from <N>] [--to <N>]
```

`--dry-run` prints the resolved plan (config, section, mappings, the stdin/files/templates/askpass blocks, the resulting command with `<secret from Title>` and `<file with Title>` placeholders) and exits without reading the store or asking for a password.

`config` after saving, and `check`, verify that the `Title`s are present in the `.kdbx` (via `keepassxc-cli export`) and offer to create the missing file/entries (`-y` — without confirmations). `check` aggregates the required `Title`s per file via `resolve` over all sections and covers all five channels (`Resolved.AllTitles` plus `Resolved.TemplateTitles`, which scans each template's content for `{{Title}}`).

## Config and cache

The config schema is the struct `Config{ Version int json:"version"; Sections map[string]Section json:"sections"; Cache *CacheConfig json:"cached,omitempty" }`; `Section{ KeyStore, Secrets map[string]string (env variable → entry name; the key is unique, so one entry can feed several variables), Stdin []string, StdinKeepOpen bool, Files []string, Templates []Template, Askpass string, AskpassReads int }`, `Template{ Name, Path string }`. When sections are merged, `Secrets` is extended entry by entry while `Stdin`/`Files`/`Templates` are replaced as a whole (the order of the lines matters). `config.Load` reads only the schema of the current version (`config.CurrentVersion`); a config without the `version` field is treated as version 0, and any version other than the current one is an error that suggests running `kdbx-cli migrate`.

**Config versioning.** The tool has been released, so the config format changes only explicitly: any incompatible schema change (renaming/inverting/removing a field, changing a type) must bump `config.CurrentVersion`, add an `N → N+1` step to `steps` (`internal/config/migrate.go`, operating on raw JSON) and cover it with a test. `config.Save` always writes the current version into the config. `kdbx-cli migrate [--from N] [--to M]` (defaults: `--from=0`, `--to=config.CurrentVersion`) applies the chain of steps to the file in place; a file already at `--to` is not an error, while a file version that does not match `--from` is.

Password caching (`internal/keyring`) is opt-in only through the `cached` section (`enabled`, `ttl`); no flags or env variables. It stores the `.kdbx` master password behind the `backend` interface: on Linux in the kernel keyring (`kernel_linux.go`, `keyctl` through `golang.org/x/sys/unix`, key `kdbx-cli:<canonical path>` from `storeKey`, TTL via `KEYCTL_SET_TIMEOUT`), on other systems in the OS keyring via `github.com/zalando/go-keyring` (`secretservice_other.go`, JSON with `expires_at`). `domain.UnlockExport(…, allowCache)` is the single entry point for "get the password (cache→prompt) + export"; the cache is read only when every requested Title is in the tool's config (`domain.ConfiguredTitles`). `keyring.New(cfg.Cache)` builds the policy. `kdbx-cli forget` (`cmdForget` → `keyring.ForgetAll`) removes every `kdbx-cli:*` key and the old Secret Service entries. Tests swap `store` (the backend) and the keyring wrappers (`keyringSet/Get/Delete/DeleteAll`). It degrades gracefully without a keyring (a miss plus a warning, no crash).

## Structure

The code is split into packages under `internal/` (no cycles: `config`/`keepass`/`term`/`secretpipe` are leaves; `keyring → config`; `domain → config,keepass,keyring,term`; `cmd → domain,config,keepass,keyring,term,secretpipe`; the root `main → cmd`).

- `cmd/kdbx-cli/main.go` (package `main`) — only `var version` (set via `-X main.version`) and the `cmd.Execute(version)` call.
- `internal/config` — `Config{Sections,Cache}`/`Section`/`CacheConfig` (JSON), `Load`/`Save` (`Load` warns about unsafe permissions through `WarnUnsafePermissions`/`PermissionWarnings`), `CurrentVersion`, `Migrate` (schema migration steps); `ExpandHome`, `Dir` (`~/.config/kdbx-cli`), `DefaultPath` (`~/.config/kdbx-cli/default`).
- `internal/keepass` — `CheckEngine` (finds `keepassxc-cli`, vets its permissions with `checkEngineFile` and pins the path in `enginePath`), `Run` (invoking the pinned `keepassxc-cli`), KeePass XML parsing (`ParseSecrets`, the `Entry` type), `LookupSecret`; write operations `CreateStore` (`db-create`), `AddEmptySecret` (`mkdir`+`add`).
- `internal/keyring` — `Cache`, `New(cfg.Cache)`, the `Get/Remember/Forget/Status` methods, `ForgetAll`, the `backend` interface (`kernelBackend` on Linux, `secretServiceBackend` elsewhere), `storeKey`, the swappable `keyringSet/Get/Delete/DeleteAll` (go-keyring).
- `internal/term` — terminal input through `/dev/tty` (`KDBX_CLI_PASSWORD` for tests): `ReadPassword`, `ReadWithPrefill` (fallback input), `Confirm` (Y/N, honours `-y`), `IsInteractive`.
- `internal/secretpipe` — secret delivery outside env: `Set.File` (an anonymous `memfd` with mode `0600`, sealed by `sealAnonFile`, handed to the child through `cmd.ExtraFiles` as `/dev/fd/N`), `Set.Askpass(name, value, reads)` (a FIFO in a `0700` directory plus a script that refuses `Username*`/`(yes/no` prompts and runs `head -n 1`; the `feed` goroutine serves the secret to the first `reads` readers, `DefaultAskpassReads` = 1 by default, and an empty answer to the rest), `Set.Close`.
- `internal/domain` — application logic: `Resolve` (merging `default` → tool section → flags, the `Overrides`/`Resolved` types, `Resolved.AllTitles`), `TemplateTitles`/`RenderTemplate`/`Resolved.TemplateTitles` (scanning a template's content for `{{Title}}` and rendering it, JSON-escaping values for `.json` paths), `Mapping`/`MappingsFromMap`/`MappingsToMap`/`SortMappings`/`ValidateEnvNames`, `UnlockExport` (cache→prompt→export), `AggregateStores`/`AggregateStoreMappings`, `gatherMissing`, `ReconcileStores` (report + creation of what is missing), `BuildStoreViews`/`StoreView`.
- `internal/cmd` — the CLI layer:
  - `execute.go` — `Execute(version)`: argv parsing, dispatch (`config`/`check`/`show`/`forget`/`migrate`/`version`/run mode), `usage`, `parseConfigArgs`.
  - `args.go` — `splitArgs` (on `--`), `parseRunFlags`, `mergeSecretsFlag` (`name:env` → `dst[env]=name`), `splitTitles`, `parseTemplatesFlag`, the `runFlags` type and its `overrides()`.
  - `harden_linux.go`/`harden_other.go` — `hardenProcess` (`PR_SET_DUMPABLE=0` at the start of `Execute`).
  - `run.go` — `cmdRun`: resolve → (if `--dry-run` → `printPlan`) → password (`titlesGranted` decides whether the cache may be used) → export → delivery over every channel (env, `stdinPayload`, `substitutePlaceholders` + `ExtraFiles`, `withAskpass`) → launching the child command, propagating the exit code.
  - `plan.go` — `printPlan` and the dry-run helpers (`describeSection`, `renderCommand`, `shellQuote`).
  - `migrate.go` — `parseMigrateArgs`, `cmdMigrate`.
  - `commands.go` — `cmdConfig` (TTY → TUI, otherwise `configFallback`; after saving — `ReconcileStores` for default), `cmdCheck`, `cmdForget`.
  - `tui.go` — the Bubble Tea model for configuring `default`: the `key-store` field with path completion (`refreshPathSuggestions`, `deleteLastPathSegment` on `alt+backspace`, `~` expansion on `tab`), a line-by-line mapping editor with hotkeys (`a`/`e`/`d`/`↑↓`/`tab`/`ctrl+s`/`esc`) and a legend.
  - `show.go` — `cmdShow`: a read-only TUI (`showTUI`) with a list of `.kdbx` files (navigation with `↑/↓`), `Enter` → open in the GUI via `openInKeePassXC` (a var, stubbed in tests); non-TTY → plain printing.

## Building and tests

Depends on `keepassxc-cli` being in PATH. Go 1.26.1. The TUI is built on Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`); password caching uses the kernel keyring through `golang.org/x/sys/unix` on Linux and `zalando/go-keyring` (godbus, no cgo) elsewhere. The build is static (`CGO_ENABLED=0`).

- `make build` — build the `./kdbx-cli` binary (version taken from `versions.txt`).
- `make unit-test` — unit tests (`go test ./...`).
- `make test` — the same as `unit-test`.
- `make bump-version` — bump `versions.txt`.

Integration tests live in the separate `kdbx-cli-tests` repository and run against released versions; this repository keeps only unit tests.

Releasing — a push to `main`/`master`, tag `v<versions.txt>` (see the `dimkarp93/install` conventions). Two platforms, identical artifacts:

- GitHub Actions — `.github/workflows/release.yml`.
- Gitea Actions — `.gitea/workflows/release.yml`: no external actions (checkout, Go installation and publishing are `run:` shell steps), the release is created through the Gitea API.
