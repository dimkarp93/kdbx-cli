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
kdbx-cli load    [--config <path>] [--key-store <path>] [--secrets=name:env,...]
                 [--stdin=name,...] [--secret-file=name,...]
                 [--template=name:path,...] [--askpass=name]
kdbx-cli migrate [--config <path>] [--from <N>] [--to <N>]
```

`--dry-run` prints the resolved plan (config, section, mappings, the stdin/files/templates/askpass blocks, the resulting command with `<secret from Title>` and `<file with Title>` placeholders) and exits without reading the store or asking for a password.

`config` after saving, and `check`, verify that the `Title`s are present in the `.kdbx` (via `keepassxc-cli export`) and offer to create the missing file/entries (`-y` — without confirmations). `check` aggregates the required `Title`s per file via `resolve` over all sections and covers all five channels (`Resolved.AllTitles` plus `Resolved.TemplateTitles`, which scans each template's content for `{{Title}}`).

## Config and cache

The config schema is the struct `Config{ Version int json:"version"; Sections map[string]Section json:"sections"; Cache *CacheConfig json:"cached,omitempty" }`; `Section{ KeyStore, Secrets map[string]string (env variable → entry name; the key is unique, so one entry can feed several variables), Stdin []string, StdinKeepOpen bool, Files []string, Templates []Template, Askpass string, AskpassReads int }`, `Template{ Name, Path string }`. When sections are merged, `Secrets` is extended entry by entry while `Stdin`/`Files`/`Templates` are replaced as a whole (the order of the lines matters). `config.Load` reads only the schema of the current version (`config.CurrentVersion`); a config without the `version` field is treated as version 0, and any version other than the current one is an error that suggests running `kdbx-cli migrate`.

**Config versioning.** The tool has been released, so the config format changes only explicitly: any incompatible schema change (renaming/inverting/removing a field, changing a type) must bump `config.CurrentVersion`, add an `N → N+1` step to `steps` (`internal/config/migrate.go`, operating on raw JSON) and cover it with a test. `config.Save` always writes the current version into the config. `kdbx-cli migrate [--from N] [--to M]` (defaults: `--from=0`, `--to=config.CurrentVersion`) applies the chain of steps to the file in place; a file already at `--to` is not an error, while a file version that does not match `--from` is.

Password caching (`internal/keyring`) is opt-in only through the `cached` section (`enabled`, `ttl`); no flags or env variables. It stores the `.kdbx` master password and the Titles allowed through `kdbx-cli load` as a JSON `entry{Password, Titles}` (`encodeEntry`/`decodeEntry`; a bare password from earlier versions decodes as an entry without Titles) behind the `backend` interface, which treats the value as opaque: on Linux in the kernel keyring (`kernel_linux.go`, `keyctl` through `golang.org/x/sys/unix`, key `kdbx-cli:<canonical path>` from `storeKey`, TTL via `KEYCTL_SET_TIMEOUT`), on other systems in the OS keyring via `github.com/zalando/go-keyring` (`secretservice_other.go`, JSON with `expires_at`). `store.UnlockExport(…, allowCache)` is the single entry point for "get the password (cache→prompt) + export"; the cache is read only when every requested Title is in the tool's config (`resolve.ConfiguredTitles`) or in `Cache.Granted`. `keyring.New(cfg.Cache)` builds the policy. `Cache.Grant` merges Titles into the live entry and restarts the TTL, `Cache.Remember` keeps the existing Titles. `kdbx-cli load` (`cmdLoad`) always prompts for the password (`UnlockExport` with `allowCache=false`), checks that every Title exists and calls `Cache.Grant`; it resolves the `default` section plus the flags and requires the `cached` section to be enabled. Grants are dropped only together with the entry. `kdbx-cli forget` (`cmdForget` → `keyring.ForgetAll`) removes every `kdbx-cli:*` key and the old Secret Service entries. `Cache` holds its dependencies in the `storage` field (the `backend` plus the legacy `keyring.Keyring` from go-keyring); `New` and `ForgetAll` use `systemStorage()` (`kernelBackend`/`secretServiceBackend` and `systemKeyring`), tests build `Cache` over fakes; there are no package-level variables to swap. It degrades gracefully without a keyring (a miss plus a warning, no crash).

## Structure

The code is split into packages under `internal/` (no cycles: `config`/`term`/`secretpipe` are leaves; `keepass` — a leaf; `keyring → config`; `tmpl → config`; `resolve → config,tmpl`; `store → config,resolve,keepass,keyring,term`; `deliver → config,secretpipe,tmpl`; `tui → config,resolve,store,keyring`; `cmd` is glue over all of them; the root `main → cmd`).

- `cmd/kdbx-cli/main.go` (package `main`) — only `var version` (set via `-X main.version`) and the `cmd.Execute(version)` call.
- Collection helpers (`Filter`, `Map`, `FlatMap`, `Unique`, `UniqueBy`, `GroupBy`, `ToSet`, `SortedKeys`, `SortedUnique`, `At`, `OrDefault`, `CountFunc`, `CountBy`) come from the shared module `github.com/dimkarp93/utils/pkg/collections` (private, vendored); there is no local `internal/utils`.
- `internal/config` — `Config{Sections,Cache}`/`Section`/`CacheConfig` (JSON), `Load`/`Save` (`Load` warns about unsafe permissions through `WarnUnsafePermissions`/`PermissionWarnings`), `CurrentVersion`, `Migrate` (schema migration steps); `ExpandHome`, `Dir` (`~/.config/kdbx-cli`), `DefaultPath` (`~/.config/kdbx-cli/default`).
- `internal/keepass` — `CheckEngine` (finds `keepassxc-cli`, vets its permissions with `checkEngineFile` and pins the path in `enginePath`), `Run` (invoking the pinned `keepassxc-cli`), KeePass XML parsing (`ParseSecrets`, the `Entry` type), `LookupSecret`; write operations `CreateStore` (`db-create`), `AddEmptySecret` (`mkdir`+`add`).
- `internal/keyring` — `Cache`, `New(cfg.Cache)`, the `Get/Remember/Grant/Granted/Enabled/Forget/Status` methods, `entry`/`mergeTitles`, `ForgetAll`, the `backend` interface (`kernelBackend` on Linux, `secretServiceBackend` elsewhere), `storeKey`, `storage`/`systemStorage`, `systemKeyring` (the go-keyring implementation of `keyring.Keyring`).
- `internal/term` — terminal input through `/dev/tty` (`KDBX_CLI_PASSWORD` for tests): `ReadPassword` (returns `ErrNoTerminal` when there is no real terminal), `ReadWithPrefill` (fallback input), `Confirm` (Y/N, honours `-y`), `IsInteractive`.
- `internal/secretpipe` — secret delivery outside env: `Set.File` (an anonymous `memfd` with mode `0600`, sealed by `sealAnonFile`, handed to the child through `cmd.ExtraFiles` as `/dev/fd/N`), `Set.Askpass(name, value, reads)` (a FIFO in a `0700` directory plus a script that refuses `Username*`/`(yes/no` prompts and runs `head -n 1`; the `feed` goroutine serves the secret to the first `AskpassReads(reads)` readers, `DefaultAskpassReads` = 1 by default, and an empty answer to the rest), `Set.Close`.
- `internal/tmpl` — templates: `Titles` (scanning a template's content for `{{Title}}`), `Render` (interpolating values, JSON-escaping them for `.json` paths).
- `internal/resolve` — `Resolve` (merging `default` → tool section → flags as layers, the `Overrides`/`Resolved` types), `Resolved.AllTitles`, `Resolved.TemplateTitles`, `ConfiguredTitles`; `Mapping`/`MappingsFromMap`/`MappingsToMap`/`SortMappings`/`ValidateEnvNames`.
- `internal/store` — work with `.kdbx` files: `UnlockExport` (cache→prompt→export), `AggregateStores`/`AggregateStoreMappings` (over `sectionUsages`, the `Channel*` labels), `ReconcileStores` (`gatherMissing` → report → creation of what is missing), `BuildStoreViews`/`StoreView`.
- `internal/deliver` — assembling the child's inputs: `InheritedEnv` (drops `KDBX_CLI_PASSWORD`), `SecretEnv`, `WithAskpass`/`AskpassEnv`, `Stdin` (`stdinPayload`), `RenderFiles` → `Files` (`Paths` gives `/dev/fd/N`, `Open` creates the memfds), `Placeholder`, `SubstitutePlaceholders`, `CloseFiles`.
- `internal/tui` — Bubble Tea models. `RunConfig` → `ConfigResult`: `configTUI` is glue over `keyStoreField` (path completion: `refreshPathSuggestions`, `deleteLastPathSegment` on `alt+backspace`, `~` expansion on `tab`), `pairList`, `pairEditor` (hotkeys `a`/`e`/`d`/`↑↓`/`tab`/`ctrl+s`/`esc`) and `cacheForm`. `RunShow(ShowData, LaunchFunc)`: a read-only `showTUI` with a list of `.kdbx` files (`↑/↓`, `Enter` → open in the GUI through `LaunchKeePassXC` or a fake in tests); `PrintStores` for non-TTY; `GrantedLine`, `FormatDuration`.
- `internal/cmd` — the CLI layer (glue):
  - `execute.go` — `Execute(version)`: `handleBuiltin` (version, completion, `--path`), `runSubcommand` (`config`/`check`/`show`/`forget`/`load`/`migrate`), `runMode`; `usage`/`usageText`, `parseConfigArgs`.
  - `args.go` — `splitArgs` (on `--`), `parseRunFlags` (`parseArg` → `setSwitch`/`setValue`), `mergeSecretsFlag` (`name:env` → `dst[env]=name`), `splitList`, `parseTemplatesFlag`, the `runFlags` type and its `overrides()`.
  - `harden_linux.go`/`harden_other.go` — `hardenProcess` (`PR_SET_DUMPABLE=0` at the start of `Execute`).
  - `secrets.go` — helpers shared by `run` and `load`: `configPathOrDefault`, `loadRunConfig`, `requestedTitles`, `existingKeyStore`, `parseEntries`, `lookupAll`, `titlesGranted`.
  - `run.go` — `cmdRun`: `resolveRun` → `runPlan` → (if `--dry-run` → `printPlan`) → `fetchSecrets` (password; `titlesGranted` decides whether the cache may be used, counting `Cache.Granted` too) → `runChild` (`prepareChild` builds `childSpec` through `deliver`, `execChild` launches it and propagates the exit code).
  - `plan.go` — `printPlan` (`printPlanHeader`/`Mappings`/`Stdin`/`Files`/`Templates`/`Askpass`) and the dry-run helpers (`describeSection`, `renderCommand`, `shellQuote`).
  - `migrate.go` — `parseMigrateArgs`, `cmdMigrate`.
  - `commands.go` — `cmdConfig` (`editDefaultSection`: TTY → `tui.RunConfig`, otherwise `configFallback`; `applyDefaultEdit`; after saving — `ReconcileStores` for default), `cmdCheck`, `cmdForget`.
  - `load.go` — `cmdLoad` (`loadAndGrant`: always prompts, validates the Titles, `Cache.Grant`), `printLoaded`.
  - `show.go` — `cmdShow`: builds `tui.ShowData`, TTY → `tui.RunShow`, otherwise `tui.PrintStores`.

## Building and tests

Depends on `keepassxc-cli` being in PATH. Go 1.27.2. The TUI is built on Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`); password caching uses the kernel keyring through `golang.org/x/sys/unix` on Linux and `zalando/go-keyring` (godbus, no cgo) elsewhere. The build is static (`CGO_ENABLED=0`).

- `make build` — build the `./kdbx-cli` binary (version taken from `versions.txt`).
- `make unit-test` — unit tests (`go test ./...`).
- `make test` — the same as `unit-test`.
- `make bump-version` — bump `versions.txt`.

Integration tests live in the separate `kdbx-cli-tests` repository and run against released versions; this repository keeps only unit tests.

Releasing — a push to `main`/`master`, tag `v<versions.txt>` (see the `dimkarp93/install` conventions). Two platforms, identical artifacts:

- GitHub Actions — `.github/workflows/release.yml`.
- Gitea Actions — `.gitea/workflows/release.yml`: no external actions (checkout, Go installation and publishing are `run:` shell steps), the release is created through the Gitea API.
