# SSH

F4 is dark's home for power-user features that live outside the Omarchy surface. The first and (in v1) only inhabitant is a complete SSH management workbench — keys, agent, client config, known hosts, authorized keys, and a read-only view of the server config. The goal: stop editing `~/.ssh/*` by hand without losing the interoperability of keeping them as real files on disk.

## What F4 SSH is for

SSH management is full of small, correct steps that are easy to botch: file modes, agent TTLs, known_hosts drift, config blocks that quietly shadow each other, forgetting which key belongs to which host. dark doesn't replace OpenSSH — it intercepts the common operations, validates the state around them, and leaves the files themselves exactly where OpenSSH expects them. You can still use plain `ssh` on the command line and dark will see your changes on the next snapshot tick.

Typical reasons to use F4 SSH:

- **Generate a new key with the right defaults.** ed25519, sensible comment, correct file modes, no `passphrase stored in a shell variable` dance.
- **See what's actually in your agent** and drop specific keys when you're done with them — or clear everything with one action.
- **Audit `~/.ssh/config`** for hosts you forgot you had, or add a new host without remembering the exact IdentityFile syntax.
- **See the fingerprint of every authorized key** that can log into this machine without grepping through the file.
- **Check whether `password authentication` is disabled on sshd** without opening `/etc/ssh/sshd_config` in a terminal.

If all you want is to type `ssh somewhere` once in a while, you don't need this. If you're rotating keys, managing multiple machines, or want an LLM to reason over your SSH state through MCP, this is the right surface.

## The six subsections

Across the top of the content pane there's a tab bar showing six sub-sections. `←`/`→` (or `h`/`l`) cycle through them. All six render the same shape: a list on the left, a detail pane on the right.

- **Keys** — every key pair under `~/.ssh`. Dots beside a key mean it's currently loaded in the agent. The detail pane shows type, bits, comment, fingerprint, and whether the private key is passphrase-protected. Private key contents are never displayed.
- **Agent** — whether `ssh-agent` is running, whether dark's managed systemd user unit is installed, and what keys the agent currently holds.
- **Client** — parsed view of `~/.ssh/config` with one entry per `Host` block. All unknown directives are preserved so round-trips don't lose data.
- **Known Hosts** — parsed `~/.ssh/known_hosts`. Hashed entries show as `(hashed)` because there's no way to recover the hostname.
- **Authorized** — `~/.ssh/authorized_keys`. Incoming connections from these keys are allowed.
- **Server** — parsed `/etc/ssh/sshd_config`. Read-only in v1.

## The mental model: files as the source of truth

dark never caches SSH state anywhere outside the daemon's in-memory snapshot. Every mutation — generate a key, add a host, load into the agent — hits real files in `~/.ssh` or invokes the real `ssh-*` tools. If you run `ssh-keygen` on the command line, dark's next snapshot tick (every 30 seconds by default) picks up the change automatically. If you delete `~/.ssh/config` with `rm`, dark will render an empty client config on the next tick.

Writes go through an atomic temp-file-plus-rename pattern with a `.bak` next to the original on the first successful write per session, so a crash mid-write can't leave you with a half-rewritten file. Permission bits are enforced on every write: 0600 for private keys and config, 0644 for `.pub` files.

## Patterns

### 1. Rotate a key with zero downtime

1. Generate a new key (e.g. `id_ed25519_2026`) from the Keys tab.
2. Copy the public key and paste it into the authorized_keys file on each machine you connect to.
3. Verify ssh works with the new key.
4. Add the new key to the agent so daily use picks it up automatically.
5. Delete the old key from the Keys tab once you're confident nothing still depends on it.

The old key's `.bak` stays in place as a safety net until you explicitly clean it up.

### 2. Add a new host and auto-populate known_hosts

From the Client tab, add a Host block with HostName, User, and IdentityFile set. On save, dark runs `ssh-keyscan` against the hostname and automatically adds the resulting entries to `~/.ssh/known_hosts`. First-connection TOFU is gone — the host key is already pinned the moment the entry lands in config.

If the host isn't reachable at the time of save (VPN down, laptop offline), the save still succeeds and the scan just gets skipped. Re-save the entry later to retry, or run a manual scan from the Known Hosts tab.

### 3. Time-limited agent keys for ephemeral sessions

When adding a key to the agent, specify a lifetime (TTL) in seconds. The agent auto-expires the key after that many seconds and you don't have to remember to remove it. Handy for:

- Production jump boxes you only touch occasionally.
- Shared machines where you don't want your key sitting in the agent forever.
- Scripted workflows that need the key for a few minutes and then should forget it.

A lifetime of `0` means no expiry — the key stays until the agent restarts or you explicitly remove it.

### 4. Agent lifecycle via systemd

dark writes `~/.config/systemd/user/ssh-agent.service` on first use. Start and stop go through `systemctl --user`, which means the agent survives terminal sessions and reconnects to the same socket across logins. Add this to your shell rc so every shell picks up the managed agent:

```sh
export SSH_AUTH_SOCK="$XDG_RUNTIME_DIR/ssh-agent.socket"
```

With that set, plain `ssh` and dark-managed `ssh-add` both talk to the same agent. No dueling agents, no lost state.

### 5. Audit sshd_config from dark

The Server tab renders the parsed sshd_config in a read-only view. It's not an edit surface — v1 is pure audit — but it's a fast way to confirm the values you care about most:

- `PasswordAuthentication no` (required for key-only login)
- `PubkeyAuthentication yes`
- `PermitRootLogin no`
- `AllowUsers` / `AllowGroups` scoping

If any of these drift away from what you expect, open `/etc/ssh/sshd_config` in `$EDITOR` the old-fashioned way. Phase 2 adds an edit flow with `sshd -t` validation and a pkexec write path so you won't have to.

## Gotchas and best practices

- **The `.bak` file is only for the first write per session.** Subsequent writes overwrite the previous `.bak`. If you made two edits in a row and want the pre-first-edit state, you need your own backup strategy (git, restic, etc).
- **dark doesn't manage `~/.ssh/config.d/`.** If you use drop-in snippets via `Include`, those live outside the parsed view. dark will show the `Include` directive in the Host block's Extras map but won't follow it.
- **Hashed known_hosts entries are preserved on disk but shown as `(hashed)` in the UI.** dark can remove them via `ssh-keygen -R <hostname>` but can't display the original hostname — that's the whole point of hashing.
- **Passphrases never touch disk.** They flow through the overlay, the NATS request, the daemon, and the `SSH_ASKPASS` helper script that lives in `$XDG_RUNTIME_DIR` for exactly as long as the `ssh-add` subprocess needs it. The helper script is then deleted.
- **Don't rely on in-dark actions for remote authorized_keys management.** dark only manages the local `~/.ssh/authorized_keys`. Adding a key there allows inbound connections TO this machine. If you want to deploy a key TO a remote machine's authorized_keys, use `ssh-copy-id` or MCP+Lua to drive the remote side.
- **Multiple dark sessions see the same state** — they all read/write the same files. Concurrent writes are protected by atomic-rename but not by a cross-process lock. If you save two hosts from two sessions at the same time, last-write-wins.

## Phase 2 (breadcrumbs)

Every subsection renders a dim "coming in phase 2" footer banner below its content so the unfinished work is visible without a separate TODO file. What's parked for phase 2:

- **Key generation and deletion dialogs** — the backend is ready; the TUI dialogs aren't wired yet.
- **Agent add/remove dialogs with passphrase prompts.**
- **Host create/edit/delete dialogs** — the parser can round-trip edits cleanly, the UI just needs forms.
- **Remove and scan dialogs for known_hosts.**
- **Add/remove dialogs for authorized_keys.**
- **Full sshd_config editing** with `sshd -t` validation and pkexec write path.
- **Alternative backends** — 1Password CLI, gnome-keyring. The Backend interface is visible in `internal/services/ssh/backend.go` with `NewOnePasswordBackend()` and `NewGnomeKeyringBackend()` returning stubs that always error with "not implemented in v1 — see phase 2 plan".

Every mutation command is already wired on the bus (`dark.cmd.ssh.*`), exposed as a Lua action (`dark.actions.ssh.*`), and exposed as an MCP tool (`ssh_*`). So even though the TUI doesn't have dialogs yet, you can drive every operation through Lua scripts or an MCP host today. The parameter schemas show up in F5 API/Lua/MCP tabs so you can see exactly what each command expects.

## Key reference

| Key          | Action                                            |
| ------------ | ------------------------------------------------- |
| `←` `→` / `h` `l` | Cycle between the six SSH subsections        |
| `↑` `↓` / `j` `k` | Walk the active list (keys, hosts, etc.)     |
| `?`          | This help                                         |
| `esc`        | Close dialogs / back out                          |

## Under the hood

- Backend interface: `internal/services/ssh/backend.go`. `OpenSSHBackend` is the only real implementation in v1.
- Subprocess calls: `ssh-keygen`, `ssh-add`, `ssh-agent`, `ssh-keyscan`. Nothing exotic — dark is a thin orchestration layer over the standard tools.
- Bus surface: every mutation is `dark.cmd.ssh.<verb>`, every snapshot is `dark.ssh.snapshot`. The catalog in `internal/bus/catalog.go` auto-exposes these as Lua actions and MCP tools with no hand-written glue.
- Safety: atomic writes + `.bak`, explicit file mode enforcement, never-display-private-key invariant, passphrase never persisted.
- Agent unit: written to `~/.config/systemd/user/ssh-agent.service` on first use, managed via `systemctl --user`.

If you want to add a new SSH operation, add the bus subject in `internal/bus/subjects.go`, a catalog entry in `internal/bus/catalog.go`, a schema in `internal/bus/schemas.go`, and a handler in `cmd/darkd/ssh.go`. F5 MCP + Lua + API tabs pick it up automatically.
