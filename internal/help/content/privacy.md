# Privacy

The privacy-adjacent settings that are typically scattered across half a dozen tools: screen lock and idle, DNS privacy (DoT / DNSSEC), firewall, SSH server, location services, Wi-Fi MAC randomization, file indexer, core-dump storage, and recent-file history. Every toggle routes through either systemd, hyprlock/hypridle config, iwd config, or `ufw`, and most writes need pkexec because they touch root-owned files.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Privacy page has four sub-sections in an inner sidebar:

1. **Screen Lock** — screensaver timeout, lock-session timeout, screen-off (DPMS) timeout, "lock on sleep" flag, and whether hyprlock has fingerprint unlock enabled
2. **Network** — current DNS server (from `resolvectl`), DNS-over-TLS state, DNSSEC state, fallback DNS, supported protocols, firewall installed/active/rules, SSH server installed/active/enabled, Wi-Fi MAC randomization mode
3. **Data** — recent-file count from `recently-used.xbel`, file indexer (localsearch / tracker3) installed/active state, core-dump storage setting, journal disk usage
4. **Location** — geoclue installed/active state

Every row is live — dark reads config and systemd state on a tick, and most rows have a keyboard shortcut to toggle or cycle the value.

## Navigating — focus and the key flow

1. Sidebar has focus on landing. `j`/`k` picks a sub-section.
2. `enter` moves focus into the content region.
3. Each sub-section has its own set of action keys (listed below).
4. `esc` backs out.

## Actions — the complete keybinding reference

### Screen Lock

- `1` — open dialog to set the screensaver timeout (seconds). Dark edits the listener block in `~/.config/hypr/hypridle.conf` that runs `screensaver` as its on-timeout action.
- `2` — set the lock-session timeout (triggers `loginctl lock-session`)
- `3` — set the screen-off (DPMS off) timeout

### Network

- `t` — cycle DNS-over-TLS through `no` → `opportunistic` → `yes`. Writes to `/etc/systemd/resolved.conf` and restarts `systemd-resolved`.
- `e` — cycle DNSSEC through `no` → `allow-downgrade` → `yes`. Same write path.
- `f` — toggle the firewall on/off. Requires `ufw` to be installed; calls `ufw enable` or `ufw disable` via dark-helper.
- `s` — toggle the SSH server. Enables or disables `sshd.service` via systemctl.
- `m` — cycle Wi-Fi MAC randomization mode through `disabled` → `once` → `network`. Writes `AddressRandomization=` in `/etc/iwd/main.conf` and restarts `iwd`. `once` randomizes per-boot; `network` randomizes per-SSID.

### Data

- `x` — clear the recent-files history. Overwrites `~/.local/share/recently-used.xbel` with an empty XBEL document. The count in the UI drops to 0 immediately.
- `i` — toggle the file indexer (localsearch-3, with tracker-miner-fs-3 as fallback). Starts or stops the corresponding systemd user service.
- `o` — cycle core-dump storage through `external` → `journal` → `none`. Writes `Storage=` in `/etc/systemd/coredump.conf`. `none` disables coredump generation entirely; `journal` stores them inside journald (subject to size limits); `external` stores them as individual files under `/var/lib/systemd/coredump/`.

### Location

- `l` — toggle geoclue. Starts or stops the `geoclue.service`. Geoclue is used by GNOME apps, Firefox, and anything else that asks "where is this machine" via the `org.freedesktop.GeoClue2` D-Bus interface.

### Universal

- `?` — open this help drawer
- `esc` — back out

## Dialogs

### Idle timeout dialogs (`1` / `2` / `3`)

Single-field dialogs that ask for a number in seconds. Dark finds the matching `listener { … }` block in `hypridle.conf` by searching for the on-timeout marker (`screensaver`, `lock-session`, or `dpms off`) and rewrites only the `timeout = N` line inside that block. Comments on the same line are preserved.

Press `enter` to commit. Dark writes the file and signals hypridle to reload. A bad input value (e.g. `0` or a negative number) disables the timeout entirely.

## Common tasks

### Enable DNS-over-TLS so your DNS queries aren't snoopable

1. Navigate to Network sub-section.
2. Press `t` once to go from `no` → `opportunistic` (tries DoT but falls back to plain DNS on failure) or twice to go to `yes` (requires DoT; fails closed if the server doesn't support it).
3. Press `e` to enable DNSSEC signing checks. `allow-downgrade` is safest; `yes` is strict.
4. The change takes effect after systemd-resolved restarts. Verify with `resolvectl status` or check the Protocols row in the Network sub-section.

### Turn on the firewall

1. Make sure `ufw` is installed (dark shows `firewall_installed=false` otherwise).
2. Navigate to Network, press `f`. The status flips from `inactive` to `active`.
3. Review the rules list — by default ufw allows outgoing and blocks incoming. Add more rules with `sudo ufw allow <port>` from a terminal (dark doesn't wrap rule management yet).

### Randomize your Wi-Fi MAC address

1. Network sub-section, press `m` until the value reads `network` (a different MAC per SSID, stable across reboots).
2. iwd restarts and the next connection uses the randomized address.
3. To reset to hardware MAC, press `m` until it reads `disabled`.

### Lock the screen after 5 minutes of idle

1. Screen Lock sub-section, `enter` to focus.
2. Arrow to the Lock row, press `2` to open the dialog.
3. Type `300` (seconds). `enter`.
4. hypridle reloads and the new timeout applies.

### Clear recent-file history before screen-sharing

1. Data sub-section, press `x`.
2. The XBEL file is overwritten with an empty document. File managers that read it (Nautilus, Dolphin, Nemo) will show an empty recents list on next launch.

### Disable core dumps entirely

Privacy angle: core dumps can contain memory contents including passwords and tokens. To turn them off:

1. Data sub-section, press `o` until the Core Dumps row reads `none`.
2. `/etc/systemd/coredump.conf` now has `Storage=none` and no new core dumps will be generated.

### Stop geoclue from reporting location

1. Location sub-section, press `l`.
2. `geoclue.service` stops. Apps that query location will get "service unavailable" until you turn it back on.

## Data sources, for the curious

- **`~/.config/hypr/hypridle.conf`** — listener-block parser for screensaver / lock / DPMS timeouts (see privacy_test.go for the test cases)
- **`resolvectl status`** — live DNS server, protocols, fallback DNS
- **`/etc/systemd/resolved.conf`** — editable config for DNSOverTLS and DNSSEC
- **`ufw status`** — firewall state and rule list
- **`systemctl is-active/is-enabled sshd`** — SSH state
- **`~/.config/hypr/hyprlock.conf`** — fingerprint unlock flag (grepped for the string `fingerprint`)
- **`/etc/iwd/main.conf`** — `AddressRandomization` key
- **`~/.local/share/recently-used.xbel`** — `<bookmark ` tag count as a proxy for recent-file count
- **`pacman -Q localsearch tracker3-miners`** — indexer installed state
- **`systemctl --user is-active localsearch-3 tracker-miner-fs-3`** — indexer live state
- **`pacman -Q geoclue`** plus `systemctl is-active geoclue` — location state
- **`/etc/systemd/coredump.conf`** — `Storage` key
- **`journalctl --disk-usage`** — journal size line (`"Archived and active journals take up 112.0M"`)

All writes that touch `/etc/` go through `dark-helper` via pkexec. There's one authentication prompt per action; dark never stores credentials.

## Known limitations

- Firewall rules are read-only in dark — you can see them but have to add/remove with `ufw` directly.
- The SSH toggle doesn't manage key authorization. If you enable SSH without setting up keys, you're password-auth-only (and ufw still needs to allow 22).
- Hyprlock fingerprint unlock detection is a substring grep for `fingerprint` in `hyprlock.conf`. If you've commented out the line, it still reports as enabled.
- Geoclue toggling requires the `geoclue` package to be installed. On a minimal install, dark shows `location_installed=false` and the toggle no-ops.
- MAC randomization is iwd-only. If you're on NetworkManager or wpa_supplicant, the `m` key writes to a config file nothing is reading.
- Clearing the recents file only touches the freedesktop XBEL file — per-app histories (browser history, zsh history, shell history) are out of scope.
