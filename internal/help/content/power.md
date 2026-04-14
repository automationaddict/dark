# Power

Everything dark knows about this machine's power state — battery, AC, thermals, CPU governor, TLP/power profiles, lid and button handlers, and the idle/DPMS timeout chain. The overview pane gives you the TL;DR; the other sub-sections drill into specific subsystems.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Power page has six sub-sections in an inner sidebar:

1. **Overview** — battery, AC, load average, uptime, suspend state, temperatures, and whatever else dark could cheaply read. The zero-configuration landing pane.
2. **Power Profile** — the active power-profiles-daemon profile (performance / balanced / power-saver) plus the Energy Performance Preference (EPP) when the running kernel/CPU supports it.
3. **CPU** — per-CPU governor, current frequency, min/max boundaries, boost state, and the platform profile if your hardware exposes one (e.g. ThinkPad `dynamic` / `low-power` / `performance`).
4. **Thermal** — every thermal zone dark found in `/sys/class/thermal/thermal_zone*`, rendered with live temperature readings and the zone's trip points when available.
5. **System Buttons** — how `systemd-logind` handles the power button, lid switch, and docked/external-power lid cases. Read from `/etc/systemd/logind.conf`.
6. **Screen & Idle** — the `hypridle` timeout chain: screensaver lock, lock session, DPMS off. Each row is editable.

The data comes from a mix of `upower` / `UPower` D-Bus, sysfs (`/sys/class/power_supply/`, `/sys/class/thermal/`, `/sys/devices/system/cpu/`), `powerprofilesctl`, and config file parsing for the logind and hypridle settings.

## Navigating — focus and the key flow

1. On landing, the inner sidebar is selected. `j`/`k` moves between sub-sections and the content pane updates.
2. Press `enter` to move focus into the content region. The active sub-section's border highlights.
3. Each content region has its own selection scheme — Profile cycles the profile list, Buttons lets you pick a handler row to edit, Screen & Idle lets you pick a timeout to edit.
4. `esc` returns focus to the sub-section sidebar, then to the main Settings sidebar.

## Actions — the complete keybinding reference

### From anywhere on the Power page

- `j` / `k` or `↑` / `↓` — move between sub-sections (in sidebar) or rows (in content)
- `enter` — focus the content / open an edit dialog / cycle the active value
- `esc` — back out one level
- `?` — open this help drawer

### Power Profile

- `P` — cycle the active `power-profiles-daemon` profile (power-saver → balanced → performance → power-saver)
- `E` — cycle the Energy Performance Preference on hardware that supports it (performance / balance_performance / default / balance_power / power)

### CPU

- `g` — cycle the CPU scaling governor for every CPU at once (performance / powersave / schedutil / ondemand / conservative, whichever the kernel offers)

### System Buttons

- `b` — open a dialog to set the handler for the currently highlighted button row. Valid values: `ignore`, `poweroff`, `reboot`, `halt`, `suspend`, `hibernate`, `hybrid-sleep`, `suspend-then-hibernate`, `lock`.

Writes go through `dark-helper` + pkexec because `/etc/systemd/logind.conf` is root-owned. After the write, dark sends `systemctl kill -s HUP systemd-logind` so the change takes effect immediately.

### Screen & Idle

- `i` — open a dialog to set the idle timeout for the selected row (screensaver, lock, screen off). Enter a number in seconds. Dark edits `~/.config/hypr/hypridle.conf` directly and tells hypridle to reload.
- `l` — toggle the `hypridle` daemon on or off. When off, **none** of the idle timers fire — no auto-lock, no screen-off, no keyboard backlight dim, no screensaver trigger. The Idle Daemon row on the Screen & Idle panel flips between `running` (green) and `stopped` (red). When turning it back on, dark invokes `omarchy-toggle-idle` so hypridle is relaunched via `uwsm-app` with the correct Wayland session environment; if `omarchy-toggle-idle` isn't available, dark falls back to a direct `hypridle` spawn detached via setsid. This is the equivalent of Omarchy's "Stop / Now locking computer when idle" toggle.

## Common tasks

### Switch to power-saver mode to extend battery life

1. Navigate to Power Profile sub-section.
2. Press `P` until the Profile line reads `power-saver`.
3. The laptop drops TDP, disables turbo boost on Intel, and scales back CPU frequency.

### Make the power button suspend instead of power off

1. Navigate to System Buttons.
2. Press `enter` to focus the content region.
3. Arrow to the `HandlePowerKey` row.
4. Press `b`. Type `suspend`. Press `enter`.
5. The `/etc/systemd/logind.conf` line is updated and logind reloads.

### Screen goes to sleep too fast

1. Navigate to Screen & Idle.
2. `enter` to focus, arrow to `screen_off`.
3. `i` to edit. Type `1800` (30 minutes). `enter`.
4. hypridle reloads and the new timeout applies to the next idle cycle.

### Stop the computer from auto-locking temporarily (for a long presentation, a movie, etc.)

1. Navigate to Screen & Idle, `enter` to focus.
2. Press `l`. The Idle Daemon row flips from `running` to `stopped`.
3. No idle timers will fire until you press `l` again. When you're done, come back and flip it on — hypridle restarts cleanly via `omarchy-toggle-idle` and your existing timeout chain resumes from the next trigger.

Note: this doesn't touch the hypridle config file, so your timeouts are preserved exactly as they were. It just gates the whole daemon. If you want to make certain specific timers longer without disabling everything, use `i` instead.

### Check why the CPU feels slow

1. Navigate to CPU sub-section.
2. Look at the Governor line — `powersave` caps frequency at the low end.
3. Press `g` to cycle through to `performance` (uncapped) or `schedutil` (kernel-default, dynamic).
4. If the issue persists, check Thermal — if a zone is near its trip point, the CPU is being throttled thermally and no governor change will help.

## Data sources, for the curious

- **`/sys/class/power_supply/*`** — battery capacity, charge state, AC online, cycle count, manufacturer
- **`UPower` D-Bus** — battery time-to-empty / time-to-full estimates
- **`/sys/devices/system/cpu/cpu*/cpufreq/`** — governor, min/max freq, current freq
- **`/sys/class/thermal/thermal_zone*/`** — zone type, current temperature, trip points
- **`powerprofilesctl`** — active profile and available profiles
- **`/sys/firmware/acpi/platform_profile*`** — platform profile on supported hardware
- **`/proc/loadavg`**, **`/proc/uptime`** — load and uptime
- **`/etc/systemd/logind.conf`** — button/lid handlers (parsed and rewritten via dark-helper)
- **`~/.config/hypr/hypridle.conf`** — the idle timeout listener blocks (parsed and rewritten directly)

## Known limitations

- EPP and CPU boost controls are per-CPU at the hardware level but dark writes to all CPUs at once. There's no per-core override.
- Thermal trip points are read but not configurable — that would require touching ACPI tables.
- The power-profiles-daemon profile list is fixed; custom profiles aren't enumerated.
- There's no battery-level notification threshold (e.g. "warn at 15%"). Use `upower --monitor` or a standalone notifier.
- Dark doesn't surface TLP, auto-cpufreq, or thermald config even when those are installed — it only talks to power-profiles-daemon.
