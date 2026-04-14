# Date & Time

Read and edit the system clock, timezone, NTP sync state, and RTC configuration. Everything routes through `timedatectl` / `systemd-timedated` on the D-Bus side, so writes are authenticated via polkit and propagate to every component that listens for clock changes.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Date & Time page has three sub-sections in an inner sidebar:

1. **Time** — the current local and UTC time, timezone, UTC offset, locale, uptime, and the 12h/24h clock format dark uses elsewhere in the UI (read from the waybar config).
2. **Sync** — NTP on/off, whether the clock has synchronized at least once, the current NTP server (from systemd-timesyncd), poll interval, measured jitter.
3. **Hardware** — RTC time as the kernel sees it, RTC date, whether the hardware clock is stored as UTC or local time.

Every read ticks on a 1-second cadence from the daemon, so the displayed time updates in real-time.

## Navigating — focus and the key flow

1. Sidebar has focus on landing. `j`/`k` picks the sub-section.
2. `enter` moves focus into the content region.
3. Action keys apply per-row changes.
4. `esc` backs out.

## Actions — the complete keybinding reference

### Time

- `z` — open the timezone picker dialog. Lists every timezone `timedatectl list-timezones` returned. Navigate with `j`/`k`, `enter` to commit.
- `t` — open the Set time dialog. Only usable when NTP is disabled (`systemd-timesyncd` will snap the clock right back otherwise). Enter a time in `YYYY-MM-DD HH:MM:SS` format.
- `f` — toggle the waybar clock format between 12h and 24h. Edits `~/.config/waybar/config.jsonc` directly and swaps `%H`/`%I` + `%p` literals in the format string.

### Sync

- `n` — toggle NTP on/off. Calls `timedatectl set-ntp <true|false>` through D-Bus.

### Hardware

- `r` — toggle the RTC's reference between UTC and local time. Calls `timedatectl set-local-rtc`. Changing this rewrites `/etc/adjtime` and is the right move if you dual-boot with Windows (which expects local-time RTC).

### Universal

- `?` — open this help drawer
- `esc` — back out

## Dialogs

### Timezone picker

A scrollable list of every timezone systemd knows about (typically 400+ entries under `/usr/share/zoneinfo/`). `j`/`k` scrolls, `enter` commits. The list is sorted alphabetically with regions grouped: `Africa/*`, `America/*`, `Asia/*`, `Europe/*`, etc. If you know what you're looking for, page-down fast — there's no incremental search yet.

On commit, dark calls `timedatectl set-timezone <zone>` via D-Bus. `systemd-timedated` writes `/etc/localtime` and pokes every listener so the change propagates immediately.

### Set time

Only enabled when NTP is disabled (the dialog reports "NTP is on — turn it off first" if you try with `n` still in sync mode). Format is strict: `YYYY-MM-DD HH:MM:SS`, 24-hour clock, local time. Press `enter` to commit — dark calls `timedatectl set-time <string>`.

## Common tasks

### Fix the timezone after moving

1. Navigate to Time, press `z`.
2. Scroll to your region (e.g. `America/New_York`, `Europe/Berlin`, `Asia/Tokyo`).
3. `enter`. The timezone updates immediately, and the Local Time display jumps to match.

### Stop Windows and Linux fighting over the clock after dual-boot

Windows assumes the RTC stores local time. Linux defaults to UTC. If you dual-boot and notice your clock is hours off after switching OSes:

1. Navigate to Hardware.
2. Press `r` to switch RTC to local time.
3. Reboot into Windows once to let it resync.

Alternative: tell Windows the RTC is UTC instead (registry edit). Both work; picking local-time RTC in Linux is easier and reversible.

### Manually set the clock for testing

1. Navigate to Sync, press `n` to turn NTP off.
2. Navigate to Time, press `t`.
3. Type e.g. `2026-12-25 09:00:00`. Press `enter`.
4. Verify the Local Time display jumps to your chosen value.
5. When done, go back to Sync and press `n` to re-enable NTP — the next poll will snap the clock back to real time.

### Switch from 24h to 12h clock display in the waybar

1. Navigate to Time.
2. Press `f`. The Clock Format row flips and the waybar config is rewritten.
3. Run `pkill -SIGUSR2 waybar` (or restart waybar) for the change to show in the bar — dark edits the file but doesn't signal waybar to reload.

### See if NTP actually synced

The Sync sub-section shows two separate lines:

- **NTP** — whether `systemd-timesyncd` is configured to run
- **Synced** — whether it has successfully fetched time from its server at least once since boot

`NTP=yes Synced=no` means timesyncd is running but hasn't contacted a server yet (or has failed to). Check the Server / Poll Interval / Jitter rows — if server is empty, check `/etc/systemd/timesyncd.conf` for a misconfigured NTP pool.

## Data sources, for the curious

- **`org.freedesktop.timedate1` D-Bus interface** — timezone, NTP state, synced state, CanNTP, LocalRTC, SetTime, SetTimezone, SetNTP, SetLocalRTC, ListTimezones
- **`org.freedesktop.timesync1` D-Bus interface** — NTP server name, poll interval, measured jitter
- **Go `time.Now()` + `.Zone()` + `.UTC()`** — local/UTC time and UTC offset formatting
- **`/sys/class/rtc/rtc0/time` and `.../date`** — RTC clock readouts from the kernel
- **`/etc/locale.conf`** — LANG environment value
- **`~/.config/waybar/config.jsonc`** — clock format substring parsed to infer 12h vs 24h
- **`/proc/uptime`** — uptime readout

## Known limitations

- No incremental search in the timezone picker. Paging through 400+ entries is slow for typists.
- The Set Time dialog requires strict `YYYY-MM-DD HH:MM:SS` format. No fuzzy parsing, no relative times.
- Clock format toggle only touches the waybar config — it doesn't signal waybar to reload, doesn't affect other clock displays (e.g. hyprlock), and doesn't surface custom waybar format strings (dark expects the default format).
- NTP server switching isn't surfaced — if you want a different pool, edit `/etc/systemd/timesyncd.conf` manually and reload timesyncd.
- Leap-second handling, NTP authentication, and chrony (as an alternative to timesyncd) aren't surfaced.
