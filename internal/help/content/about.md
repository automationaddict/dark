# About

The **About** tab gives you a live snapshot of the machine dark is running on. Everything you see is read when you open the tab and refreshed every two seconds, so uptime and memory numbers move while you watch.

## System

Identity of the host itself.

- **Host** — the kernel hostname, via `os.Hostname()`.
- **OS** — `PRETTY_NAME` from `/etc/os-release`, which is how your distribution identifies itself.
- **Kernel** — the running kernel version from `uname -r`.
- **Arch** — machine architecture from `uname -m`.
- **Uptime** — how long the system has been up, parsed from `/proc/uptime`.

## Hardware

What's inside the box.

- **CPU** — model name from the first `processor` block in `/proc/cpuinfo`.
- **Cores** — count of logical processors (hardware threads), not physical cores.
- **Memory** — used vs total. "Used" is `MemTotal - MemAvailable`, which matches what most system monitors show and reflects user-facing memory pressure.
- **Swap** — used vs total swap from `/proc/meminfo`.

## Session

Details about the current user session and desktop environment.

- **User** — current username from `$USER`.
- **Shell** — login shell from `$SHELL`, basename only.
- **Terminal** — `$TERM_PROGRAM` if set, otherwise `$TERM`.
- **Desktop** — detected from `$XDG_CURRENT_DESKTOP`. If that's unset but `$HYPRLAND_INSTANCE_SIGNATURE` exists, dark reports `Hyprland` directly.

## dark

Information about the running panel itself.

- **Go** — Go toolchain version dark was built with, from `runtime.Version()`.
- **Binary** — absolute path of the running `dark` binary.
- **Built** — modification time of that binary. For dev builds that's effectively "the last time you rebuilt".

## Refresh

The panel refreshes every 2 seconds via a `tea.Tick`. Pressing `ctrl+r` from inside dark rebuilds and re-exec's the binary in place, which also resets the Built timestamp.

## Tips

- `j` / `k` or arrow keys move between sidebar sections.
- `+` / `-` resize the sidebar.
- `ctrl+r` rebuilds and hot-reloads the running panel.
- `?` opens or closes this help drawer.
