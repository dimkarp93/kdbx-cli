# Kernel keyring

On Linux `kdbx-cli` caches the `.kdbx` master password in the **kernel keyring**, a kernel subsystem that keeps small secrets in kernel memory. It is neither a daemon nor a file: the password never touches the disk and never travels over D-Bus.

## How the password is stored

- **Where:** the user keyring (`@u`). It is shared by all of the user's processes and lives until reboot.
- **Key:** type `user`, name `kdbx-cli:<absolute path to the .kdbx with symlinks resolved>`, data — the password.
- **Lifetime:** `KEYCTL_SET_TIMEOUT` = the config's `ttl`. When it runs out, the kernel removes the key, not `kdbx-cli`.
- **Permissions:** `3f010000` — the owner has every right, everyone else may only view metadata.

The code is `internal/keyring/kernel_linux.go`. System calls go through `golang.org/x/sys/unix` (`AddKey`, `KeyctlSearch`, `KeyctlBuffer`, `KeyctlInt`), without cgo.

## The `keyctl` utility

Needed only for manual inspection; `kdbx-cli` works without it.

```sh
sudo apt install keyutils
sudo pacman -S keyutils
sudo dnf install keyutils
```

```sh
keyctl show @u
keyctl search @u user kdbx-cli:/home/me/store.kdbx
keyctl print <id>
keyctl timeout <id> 600
keyctl invalidate <id>
```

| Command | What it does |
|---|---|
| `keyctl show @u` | List the user's keys: id, permissions, type, name |
| `keyctl search @u user <name>` | Key id by name |
| `keyctl print <id>` | The key's data, i.e. **the password in plain text** |
| `keyctl timeout <id> <sec>` | Set a new lifetime |
| `keyctl invalidate <id>` | Remove the key right away |

To remove every `kdbx-cli` key, `kdbx-cli forget` is simpler.

## `/proc/keys`

`keyctl` can set a key's lifetime but cannot read it back. The remaining time is visible only in `/proc/keys`, a virtual file the kernel builds on every read (`proc_keys_show()` in `security/keys/proc.c`). `expiresIn` reads it so that `show` can tell how long the cache will live.

```sh
grep kdbx-cli /proc/keys
```

```
1b2c3d4e I--Q---     1   9m 3f010000  1000  1000 user      kdbx-cli:/home/me/store.kdbx: 12
```

| Field | Example | Meaning |
|---|---|---|
| 1 | `1b2c3d4e` | key id, hex, always 8 characters (`%08x`) |
| 2 | `I--Q---` | flags |
| 3 | `1` | reference count |
| 4 | `9m` | **remaining time** |
| 5 | `3f010000` | permissions |
| 6, 7 | `1000 1000` | uid, gid |
| 8 | `user` | type |
| 9… | `kdbx-cli:…: 12` | name and data size |

The remaining time is one number and a unit, rounded down:

| Remaining | Output |
|---|---|
| no expiry | `perm` |
| expired | `expd` |
| under a minute | `45s` |
| under an hour | `9m` |
| under a day | `3h` |
| under a week | `2d` |
| more | `5w` |

`9m` means "between 9 and 10 minutes"; there is no finer value. `kdbx-cli` treats `perm` and `expd` as no cache.

## Limitations

- Any process running as the same user can read the password (`keyctl print`).
- `/proc/keys` lists only keys the reader may view. Your own keys are always listed.
- Without `/proc` (in some containers) `show` cannot display the remaining time, but the cache still works.
- If the kernel keyring is unavailable, the cache is off and the password is asked as usual.
- `/proc/keys` is a text interface, not an ABI structure. On any format mismatch `kdbx-cli` does not crash; it assumes there is no cache.
