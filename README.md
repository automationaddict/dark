# dark

A keyboard-driven settings panel for [Omarchy](https://omarchy.org) (Arch Linux + Hyprland), written in Go.

`dark` is a Bubble Tea TUI split across three binaries:

- **`dark`** — the interactive terminal UI.
- **`darkd`** — a user-scope daemon that owns service state and publishes snapshots over an embedded NATS bus.
- **`dark-helper`** — a small privileged helper invoked via `pkexec` for the handful of operations that require root (package installs, firmware updates, Limine snapshot management, etc.).

Nothing is installed outside `$HOME`. Privilege escalation is delegated to `dark-helper` at runtime, so the install itself never requires `sudo`.

## Requirements

- Linux on `x86_64` (no ARM builds yet).
- An Omarchy / Arch-based system with Hyprland.
- `polkit` for the `pkexec` prompt when privileged actions run.
- `systemd` user session (for the supervised `darkd` unit).

## Install

One-liner:

```sh
curl -fsSL https://raw.githubusercontent.com/automationaddict/dark/main/install.sh | bash
```

The installer downloads the latest release tarball from GitHub, verifies it against `SHA256SUMS`, and places the binaries and unit file under `$HOME`:

```
~/.local/bin/dark
~/.local/bin/darkd
~/.local/bin/dark-helper
~/.config/systemd/user/darkd.service
```

It then enables and starts the `darkd` user unit.

### Pinning a version

```sh
curl -fsSL https://raw.githubusercontent.com/automationaddict/dark/main/install.sh | bash -s -- --version v0.1.0
```

### Installer options

| Flag | Purpose |
|---|---|
| `--version <tag>` | Install a specific release tag (default: latest). |
| `--skip-unit` | Do not write the systemd user unit. |
| `--skip-enable` | Write the unit but do not enable or start it. |
| `--force` | Reinstall even if the requested version is already present. |
| `-h`, `--help` | Show usage. |

### PATH

If `~/.local/bin` is not already on your `PATH`, add it to your shell profile:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

## Running

Launch the TUI from any terminal:

```sh
dark
```

`darkd` runs in the background as a systemd user service. You can inspect it with:

```sh
systemctl --user status darkd
journalctl --user -u darkd -f
```

## Updating

`dark` ships a built-in self-update flow. Inside the TUI:

1. Press `F2` to open the **Updates** tab.
2. Select the **Dark** section in the inner sidebar.
3. Press `c` to check GitHub for the latest release.
4. If an update is available, press `u` to download, verify, and install it.

The update is an atomic binary swap under `~/.local/bin` — Linux keeps the old inode alive for the running process, and `systemctl --user restart darkd` picks up the new code.

You can also re-run the installer at any time to upgrade:

```sh
curl -fsSL https://raw.githubusercontent.com/automationaddict/dark/main/install.sh | bash
```

## Building from source

```sh
git clone https://github.com/automationaddict/dark.git
cd dark
go build ./cmd/dark
go build ./cmd/darkd
go build ./cmd/dark-helper
```

The repository uses a standard Go module layout — no code generation or external build steps are required.

## Uninstall

```sh
systemctl --user disable --now darkd
rm -f ~/.local/bin/{dark,darkd,dark-helper}
rm -f ~/.config/systemd/user/darkd.service
rm -rf ~/.cache/dark ~/.config/dark
```

## License

See `LICENSE` (to be added).
