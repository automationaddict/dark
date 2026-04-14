# System Updates

The F2 → Updates sidebar entry swaps the App Store content pane for a two-part view covering system-level updates: **Omarchy** (the system pacman upgrade path including keyring handling, orphan cleanup, and release-channel switching) and **Firmware** (fwupd-managed device firmware updates). Both use the same underlying privileged helper so a single pkexec prompt covers the whole flow.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Updates pane has an inner sidebar with two entries:

1. **Omarchy** — current version, available version (when a newer one exists), current release channel (stable / rc / edge), status ("System is up to date" / "Update available" / "Updating…"), and a progress section that populates with a per-step checklist when an update is running
2. **Firmware** — every device fwupd knows about, grouped by whether an update is available, plus the current version and the update action when one exists

To get here from anywhere: press `F2` to land on the App Store, then press `u` (or navigate in the sidebar past the last App Store category) to reach Updates.

## How Omarchy updates work

An Omarchy "full update" isn't just `pacman -Syu` — it runs a sequence of privileged steps in a single pkexec session so you only authenticate once:

1. **Sync the clock** — `systemctl restart systemd-timesyncd`. Having a working clock avoids keyring verification failures from skewed times.
2. **Update the keyring** — check whether the omarchy GPG key is already in `pacman-key --list-keys`. If not, `pacman-key --recv-keys` + `--lsign-key`, then `pacman -Sy` and install `omarchy-keyring`. Finally `pacman -Sy archlinux-keyring` for the archlinux-maintained keys.
3. **Update the system** — `pacman -Syyu --noconfirm` (double-y to force refresh even on same-timestamp databases).
4. **Remove orphans** — `pacman -Qtdq` lists every explicit package dropped as a dependency, then `pacman -Rs --noconfirm` on each one in best-effort mode (failures are silently skipped because orphan chains sometimes can't be cleanly removed).

Each step's success or failure is reported in the Progress box. If any step fails, the update stops and the status line shows the error.

## How firmware updates work

Dark shells out to **fwupd** via the `fwupdmgr` CLI for device enumeration and update application. The sequence per-device:

1. `fwupdmgr refresh` — pull the latest metadata from the LVFS (Linux Vendor Firmware Service) server
2. `fwupdmgr get-updates` — list devices with pending firmware updates
3. For each update the user confirms: `fwupdmgr update <device-id>` with the pkexec helper wrapping the privileged bits

Dark's view shows the device name, current firmware version, and whether an update is pending. The actual download and flash happens via fwupd; dark just surfaces the status.

## Release channels

Omarchy publishes three release channels via different pacman mirror URLs:

- **stable** — the default, most-tested channel. `Server = https://stable-mirror.omarchy.org/$repo/os/$arch` and `Server = https://pkgs.omarchy.org/stable/$arch` for the omarchy repo.
- **rc** — release candidate. Gets changes a week or two before stable.
- **edge** — the bleeding edge. Updated continuously from the dev branch.

Switching channels rewrites `/etc/pacman.d/mirrorlist` and the `[omarchy]` server line in `/etc/pacman.conf`. The next `pacman -Sy` pulls from the new channel.

## Navigating — focus and the key flow

1. On landing (assuming you got here via `u` from the App Store tab), focus is on the inner sidebar — Omarchy / Firmware.
2. `j`/`k` switches between Omarchy and Firmware views.
3. Action keys apply to the currently visible view.

## Actions — the complete keybinding reference

### Omarchy

- `u` — run the full update flow. Opens a confirmation dialog, then dispatches the sequence described above. The Progress box fills in as each step completes.
- `c` — open the Change channel dialog. Three-option select: stable / rc / edge. On commit, dark rewrites the mirror files.

### Firmware

- `i` — install the available firmware update for the highlighted device. Dark confirms and then runs `fwupdmgr update <device-id>` through dark-helper.

### Universal

- `?` — open this help drawer
- `esc` — back out to the App Store view

## Dialogs

### Full update confirmation

Zero-field dialog: `Run full system update?`. `enter` to confirm. The update is non-cancellable once started — pressing `esc` after confirm only closes the dialog; the daemon keeps running the sequence. Progress appears in the Progress box as each step finishes.

### Change channel

Single-field select: `stable`, `rc`, or `edge`. `enter` commits and rewrites the mirror files via dark-helper. A reboot isn't required, but you should run an update afterwards to pull packages from the new channel.

### Firmware update confirmation

Zero-field dialog: `Install firmware update for <device-name>?`. Confirm to dispatch the fwupd update. The device may need to reboot after the flash completes — fwupd reports that in the output.

## Common tasks

### Run the full system update

1. F2 → `u` to reach Updates → Omarchy.
2. Check the Current / Available versions. If they differ, there's an update.
3. Press `u`. Confirm the dialog.
4. Watch the Progress box fill in:
   - `✓ Syncing time…`
   - `✓ Updating keyring…`
   - `✓ Updating system packages…`
   - `✓ Removing orphan packages…`
5. If any step fails, the Progress box shows `✗` with the error text.
6. When done, the status line may show `⚠ Reboot recommended` — kernel or glibc updates benefit from a reboot but don't require one.

### Switch to the edge channel

1. Navigate to Updates → Omarchy.
2. Press `c`. Select `edge`. Confirm.
3. The mirror files are rewritten.
4. Run a full update (`u`) to pull the new packages.

### Install a firmware update

1. Navigate to Updates → Firmware.
2. Scroll the device list for any marked "Update available".
3. Highlight the device, press `i`. Confirm.
4. fwupd downloads the firmware, verifies signatures, and flashes. Some devices prompt for a power-cycle during the flash (BIOS/UEFI updates typically reboot automatically).

### Check if there's an update without running one

Just navigate to Updates → Omarchy and look at the Current vs Available version rows. The daemon polls for updates on a periodic tick (every ~60 seconds) and the status line reflects the latest check.

Firmware works the same way — navigate to Updates → Firmware and the list is already populated.

## Data sources, for the curious

- **`pacman -Q omarchy`** / version files — current Omarchy version
- **Omarchy release API** — pulls the latest version from the channel's mirror for the "Available version" line
- **`pacman-key --list-keys <keyid>`** — check whether the omarchy key is trusted
- **`pacman -Qtdq`** — orphan list (used for cleanup)
- **`/etc/pacman.d/mirrorlist` and `/etc/pacman.conf`** — rewritten on channel change
- **`fwupdmgr refresh` and `fwupdmgr get-updates`** — firmware update state
- **`fwupdmgr update <id>`** — the actual flash action

All privileged steps run through `dark-helper` via pkexec. The full update uses a single helper invocation that chains all four sub-commands, so you only get one pkexec prompt per update.

## Known limitations

- The full update flow isn't cancellable mid-run. Once you confirm, you have to wait for the sequence to finish (or crash).
- Per-package exclusions aren't supported. If `pacman -Syu` needs to confirm something, it won't — dark uses `--noconfirm`, which means conflicts auto-resolve in the default direction.
- AUR packages aren't handled by the Omarchy full update — that's the App Store's upgrade path (press `U` on the App Store tab for AUR upgrades via paru/yay).
- Firmware updates that require a reboot to apply don't re-run dark after boot. You have to come back to this page to confirm the update landed.
- There's no rollback for firmware — fwupd only installs newer versions. If a firmware update breaks something, recovery is device-specific.
- The release-channel switch doesn't automatically downgrade packages. If you switch from edge back to stable, packages newer than stable stay installed until the next `pacman -Syu` with `--overwrite` or an explicit downgrade.
