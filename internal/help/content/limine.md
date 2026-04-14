# Limine

Manage the [Limine bootloader](https://limine-bootloader.org) the way Omarchy ships it: browse the boot-menu snapshot entries `limine-snapper-sync` has written, trigger new snapshots, edit the three on-disk config files Limine touches, and patch the kernel command line. Every write goes through `dark-helper` so the single pkexec prompt covers all of it.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Limine page is one of the F3 → Omarchy sub-tabs. The sidebar on the left has four entries for this section:

1. **Snapshots** — the boot-menu entries `limine-snapper-sync` has generated from your Btrfs snapper snapshots
2. **Boot Config** — the top-level key/value pairs in `/boot/limine.conf`
3. **Sync Config** — `/etc/limine-snapper-sync.conf`, which controls how snapper snapshots are turned into boot entries
4. **Omarchy Defaults** — `/etc/default/limine` plus the `KERNEL_CMDLINE[default]+=` lines the Omarchy install script uses to rebuild the boot config

All four are read-only until you press `enter` to move focus into the content region; the active sub-section's border lights up in accent blue.

## The Snapshots table

This is NOT the raw list of snapper snapshots on your Btrfs filesystem. It's the subset `limine-snapper-sync` has written into `/boot/limine.conf` as bootable entries — capped by `MAX_SNAPSHOT_ENTRIES` in `/etc/limine-snapper-sync.conf`. On a stock Omarchy install that cap is `5`, so you'll see the five most recent snapshots no matter how many snapper itself is keeping.

Columns:

- **#** — the snapper snapshot number (the subvolume index under `/.snapshots/`). `-` if dark couldn't parse it out of the cmdline.
- **Timestamp** — the label from `limine.conf`, usually `YYYY-MM-DD HH:MM:SS` written by snapper at creation time.
- **Type** — `timeline` (scheduled), `pre` / `post` (pacman transaction bracket), or `single` (manual).
- **Kernel** — the kernel version the snapshot boots. Matters after a kernel upgrade if you need to roll back to a working kernel.
- **Subvol** — the Btrfs subvolume path, e.g. `/@/.snapshots/98/snapshot`.

The row marker shows which entry is selected when the content region has focus.

Below the table you'll see a summary line: `N snapshot entries · default_entry M`. `default_entry` is which index in `limine.conf` Limine boots into if you don't pick from the menu — typically `0`, which is your live root.

### Why only a handful of snapshots show up

The cap is set by `MAX_SNAPSHOT_ENTRIES` in the Sync Config pane (and mirrored in Omarchy Defaults). Snapper keeps many more snapshots than the boot menu displays — usually dozens or hundreds — but the boot menu would become unusable with that many entries, and each boot entry costs a small amount of space in `/boot`. `limine-snapper-sync` prunes down to the most-recent N whenever it runs.

If you want more bootable snapshots, bump `MAX_SNAPSHOT_ENTRIES` in Sync Config and run `s` to re-sync. The raw Btrfs cost of those additional entries is almost nothing (CoW snapshots only grow as files diverge from the live tree), so the real limit is how long a boot menu you're willing to scroll.

## Boot Config

`/boot/limine.conf` is the file Limine itself reads at boot. Dark surfaces the top-level settings:

- **timeout** — seconds to wait on the menu before booting `default_entry`. `0` boots immediately, `-1` waits forever.
- **default_entry** — index of the menu entry to boot. `0` is usually the live root.
- **interface_branding** — the text shown above the menu, e.g. `Omarchy Linux`.
- **hash_mismatch_panic** — whether to halt if a kernel/initrd hash doesn't match. `yes` on a secure setup.
- **term_background / backdrop / term_foreground** — menu colors.
- **editor_enabled** — `yes` allows pressing `e` at the menu to edit an entry before booting.
- **verbose** — `yes` prints Limine's own diagnostics during boot.

Press `enter` on a row to open a single-field dialog with the current value pre-filled. Press `enter` again to commit — dark writes back through `dark-helper` (pkexec) which edits the file in place.

## Sync Config

`/etc/limine-snapper-sync.conf` is the config for the systemd timer/service that syncs snapper snapshots into `limine.conf`. Editable keys:

- **TARGET_OS_NAME** — the label prefix for generated entries.
- **ESP_PATH** — where the EFI system partition is mounted, usually `/boot`.
- **LIMIT_USAGE_PERCENT** — don't create entries if `/boot` free space falls below this percent.
- **MAX_SNAPSHOT_ENTRIES** — the cap discussed above. Bump this to see more bootable history.
- **EXCLUDE_SNAPSHOT_TYPES** — skip snapshot types (e.g. `pre` or `post`) when generating entries. Leaving it empty includes everything snapper has.
- **ENABLE_NOTIFICATION** — `yes` for desktop notifications when the sync runs.
- **SNAPSHOT_FORMAT_CHOICE** — which built-in template decides the entry label. `0` is the default.

After editing any of these, run `s` in the Snapshots tab to re-generate `limine.conf` immediately instead of waiting for the next timer fire.

## Omarchy Defaults

`/etc/default/limine` is Omarchy's tooling config. Separate from `limine.conf` — the Omarchy install script reads this file and regenerates `/boot/limine.conf` from scratch. Editable scalar keys:

- **TARGET_OS_NAME** — label used when Omarchy generates new entries.
- **ESP_PATH** — EFI mount path.
- **ENABLE_UKI** — `yes` to use a Unified Kernel Image (signed, single-file kernel+initrd+cmdline).
- **CUSTOM_UKI_NAME** — filename for the UKI under `/boot/EFI/`.
- **ENABLE_LIMINE_FALLBACK** — `yes` to always keep an unsigned fallback entry alongside the UKI.
- **FIND_BOOTLOADERS** — `yes` makes Limine detect other installed OSes.
- **BOOT_ORDER** — how to order detected entries in the menu.
- **MAX_SNAPSHOT_ENTRIES** — mirrored from Sync Config so both scripts see the same cap.
- **SNAPSHOT_FORMAT_CHOICE** — same as Sync Config.

There's one additional virtual row at the bottom: **kernel_cmdline**. This edits the `KERNEL_CMDLINE[default]+=` lines, which Omarchy appends to the kernel command line each time it regenerates `limine.conf`. Press `enter` on it to open a multi-field dialog, one line per existing entry plus two blank rows for additions. Empty fields are dropped on save.

## Actions — the complete keybinding reference

### From the sidebar (before you've pressed enter)

- `j` / `k` or `↑` / `↓` — move between Limine sub-sections (Snapshots, Boot Config, Sync Config, Omarchy Defaults)
- `enter` — move focus into the content region
- `?` — open this help drawer
- `esc` — back out to the Omarchy-level sidebar

### From the Snapshots content region

- `j` / `k` — move the selection through the snapshot list
- `a` — create a new snapper snapshot. Opens a description dialog.
- `s` — run `limine-snapper-sync` now. Regenerates `/boot/limine.conf` from whatever snapper currently has.
- `d` — delete the selected snapshot. Runs `snapper delete <N>` which removes the Btrfs subvolume and the `limine.conf` entry on the next sync.
- `esc` — return focus to the sidebar

### From any Config content region (Boot / Sync / Omarchy)

- `j` / `k` — move the row selection
- `enter` — open an edit dialog for the selected row, pre-filled with the current value. Press `enter` again to commit or `esc` to cancel.
- `esc` — return focus to the sidebar

## Dialogs

Config edits and the create/delete/sync flows all use the standard modal dialog component.

- `tab` / `↓` — next field
- `shift+tab` / `↑` — previous field
- `backspace` — delete the previous character
- Any printable key types into the active field
- `enter` — submit
- `esc` — cancel

Because every write touches files under `/boot`, `/etc/default/`, or `/etc/limine-snapper-sync.conf`, every commit goes through pkexec via `dark-helper`. You'll get one authentication prompt per action.

## Common tasks

### Create a snapshot before doing something risky

1. Navigate to F3 → Omarchy → Limine → Snapshots.
2. Press `enter` to focus the content region.
3. Press `a`. Type a description ("before nvidia driver swap", "pre-upgrade 2026-04-14", etc.).
4. Press `enter`. Snapper creates a `single`-type snapshot. Run `s` to immediately have it appear as a boot entry, or wait for the next `limine-snapper-sync` timer fire.

### Roll back to a pre-pacman snapshot

1. Reboot and pick the snapshot entry from the Limine menu (not from dark — dark only lets you browse, not actually boot from them).
2. Once booted into the snapshot, you'll be running a read-only view of that older root. To make the rollback permanent, follow the snapper rollback procedure (`snapper rollback`) from inside that snapshot. Dark doesn't automate this because getting it wrong leaves an unbootable system.

### See more bootable snapshots in the menu

1. Navigate to Sync Config.
2. Press `enter` to focus, arrow to `MAX_SNAPSHOT_ENTRIES`, press `enter` to edit.
3. Type a larger number (15 and 20 are both comfortable, 100 is not).
4. Press `enter` to commit.
5. Switch back to Snapshots and press `s` to re-sync so `/boot/limine.conf` reflects the new cap immediately.

### Add a kernel parameter for every boot

1. Navigate to Omarchy Defaults.
2. Press `enter` to focus, arrow down past the scalar keys to the `kernel_cmdline` virtual row, press `enter`.
3. The multi-field dialog opens with one field per existing cmdline line. Fill in one of the blank trailing fields with the new parameter (e.g. `nvidia_drm.modeset=1`).
4. Press `enter` to commit. Dark writes the change back to `/etc/default/limine`.
5. The change doesn't take effect until Omarchy regenerates `limine.conf` — run the Omarchy install/upgrade flow or `s` in Snapshots (if your sync config re-reads `/etc/default/limine`, which depends on your Omarchy version).

### Delete an old snapshot to free space

1. Go to Snapshots, focus the content region, highlight the snapshot.
2. Press `d`. Confirm the dialog.
3. Snapper deletes the subvolume. Run `s` to prune its entry out of `limine.conf`.

Note: Btrfs snapshots are copy-on-write. Deleting a snapshot only frees bytes that are *exclusively* held by that snapshot — shared data stays. For a fresh snapshot this is almost nothing; for one that's been around during heavy file churn it can be hundreds of MB to a few GB.

## Status line meanings

When the daemon is working on something, dark shows a status line above the content:

- `working…` (accent) — a create/sync/delete/edit command is in flight
- `action failed: <message>` (red) — the last action returned an error, usually from `snapper`, `limine-snapper-sync`, or the filesystem. Hit `esc` to dismiss.

## Data sources, for the curious

The limine service reads:

- **`/boot/limine.conf`** — parsed with a hand-rolled walker that understands the slash-depth menu grammar. Snapshot entries come from the `//Snapshots` subtree.
- **`/etc/default/limine`** — shell-style assignment parser that handles `KEY=value` scalar rows and `KERNEL_CMDLINE[default]+=` accumulation.
- **`/etc/limine-snapper-sync.conf`** — same shell-style parser.
- **`snapper list --type single|pre|post|timeline`** — to cross-reference subvolume IDs against the boot entries. Requires read permission on the snapper config; root is sufficient, regular users need a snapper ALLOW_USERS entry.

All writes go through `dark-helper` via pkexec, which does its own path validation before touching any of these files — no arbitrary writes are possible even if darkd misbehaves.

Dark publishes a fresh limine snapshot on `dark.limine.snapshot` every time a command completes and on a periodic tick so the view stays current if another tool (or the systemd timer) changes the files underneath it.

## Known limitations

- Dark can't actually boot a snapshot — that has to happen from the Limine menu itself. The Snapshots table is browse-only.
- There's no "rollback to this snapshot" shortcut. Doing that correctly requires booting the snapshot first and running `snapper rollback`, and automating it wrong is dangerous enough that it's off-scope for now.
- The `default_entry` field in Boot Config is a raw index, not a dropdown. You need to count entries in your menu to know what number to type.
- Snapper configs other than `root` (e.g. a separate `/home` snapper config) aren't surfaced — dark only looks at the root config.
