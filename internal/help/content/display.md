# Displays

Manage connected monitors: see resolution, refresh rate, scale, rotation, and position for every output Hyprland knows about. Rearrange monitors visually, switch modes, toggle DPMS, and identify which physical screen is which.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

Dark talks to Hyprland via `hyprctl` to read and write monitor state. The page shows a single group box listing every connected monitor in a table with columns for Name, Resolution, Scale, Rotation, and Position.

Row markers:

- `▸` (accent) marks the selection cursor when the content region has focus.
- `●` (green) means the monitor is enabled and DPMS is on.
- `◌` (gold) means the monitor is enabled but DPMS is off (screen blanked).
- `○` (dim) means the monitor is disabled.

Tags beneath each row show additional state: whether the monitor is focused, VRR enabled, DPMS off, disabled, mirroring another output, or which workspace is active on it.

## Navigating

1. Press `enter` to move focus into the content region. The Monitors box border lights up in accent.
2. `j` / `k` moves the selection between monitors.
3. `enter` again drills into the monitor info panel.
4. `esc` backs out — from info panel to monitor list, from monitor list to sidebar.

## Actions

From display content focus:

- `j` / `k` — move the selection between monitors
- `enter` — open the detailed info panel for the highlighted monitor
- `m` — open the mode picker to switch resolution and refresh rate
- `r` — cycle rotation: Normal → 90° → 180° → 270° → Normal
- `+` / `=` — increase scale by 0.25 (max 4.0)
- `-` / `_` — decrease scale by 0.25 (min 0.25)
- `w` — toggle DPMS (turn the screen on or off without disabling the output)
- `e` — toggle the monitor between enabled and disabled
- `v` — toggle VRR (variable refresh rate, also known as adaptive sync or FreeSync)
- `p` — open position dialog to set the monitor's X/Y coordinates in the virtual desktop
- `l` — open the visual arrangement view to see and rearrange monitor positions graphically
- `i` — identify monitors by flashing a number on each physical screen for 3 seconds
- `esc` — return focus to the sidebar (or close the current panel)

## The Mode Picker

Press `m` to open the mode picker. It shows every resolution and refresh rate the monitor supports, with the current mode highlighted. Use `j` / `k` to move the highlight and `enter` to apply the selected mode. Press `esc` to cancel without changing anything.

Mode strings look like `1920x1080@60.00Hz`. The monitor advertises which modes it supports via EDID — if a mode you want is missing, it means the display didn't report it as a supported timing.

## The Monitor Info Panel

Press `enter` on a highlighted monitor to expand it. The monitor list is replaced by a single info panel showing:

- **Name / Description / Make / Model / Serial** — identifiers from the display's EDID data.
- **Resolution / Refresh Rate / Scale / Transform / Position** — current live settings.
- **Display Power / VRR / Status** — runtime state flags.
- **Mirror Of / Workspace** — relationship to other monitors and Hyprland workspaces.
- **Available Modes** — every timing the display reports. The current mode is highlighted.

Action keys still work while the info panel is open. Press `esc` to back out to the monitor list.

## The Arrangement View

Press `l` to open the visual layout. Monitors are drawn as proportionally scaled rectangles positioned to reflect their real pixel coordinates, like the display arrangement panel in Windows or GNOME Settings.

Controls in arrangement mode:

- `←` `→` `↑` `↓` — nudge the selected monitor's position. The monitor snaps to edges of adjacent monitors when it gets close, aligning automatically for common side-by-side and stacked arrangements.
- `j` / `k` — cycle the selection between monitors.
- `i` — identify (flash numbers on physical screens).
- `esc` — close the arrangement view and return to the monitor list.

Position changes apply immediately via `hyprctl keyword monitor` and are also written to your Hyprland config so they persist across restarts.

## Identify

Press `i` to flash a large number and name on each physical screen for 3 seconds. Useful when you have multiple monitors and need to figure out which Hyprland output name (like `eDP-1` or `DP-2`) corresponds to which physical display. The notification appears via Hyprland's built-in notification system — no extra tools needed.

## DPMS vs Disable

Two ways to turn off a screen:

- **DPMS off** (`w`) blanks the display but keeps it in Hyprland's layout. Windows on that monitor stay where they are. The monitor can be woken up instantly.
- **Disable** (`e`) removes the output from Hyprland entirely. Windows are migrated to remaining monitors. Re-enabling brings the monitor back but windows don't automatically move back.

Use DPMS when you want to temporarily blank a screen (like turning off a TV when not watching). Use disable when you want to fully remove an output from the layout.

## Config Persistence

When you change a monitor's resolution, scale, rotation, position, or VRR, dark writes the updated configuration to `~/.config/hypr/monitors.conf` so your settings survive Hyprland restarts. The file uses standard Hyprland monitor syntax:

```
monitor = eDP-1, 1920x1080@60.00, 0x0, 1.00
```

Dark preserves any comments in the file and only updates lines for monitors you've changed. If a monitor doesn't have an existing line, one is appended.

Runtime-only changes like DPMS toggle and identify are not persisted — they're inherently temporary.

## Data source

Everything on this page comes from `darkd`, which shells out to `hyprctl monitors -j` to read the current state. Commands that modify settings use `hyprctl keyword monitor` or `hyprctl dispatch dpms` under the hood. The daemon publishes monitor snapshots on `dark.display.monitors` every 10 seconds, with immediate republish after any action command.

## Limitations

- Hyprland doesn't expose a display change event subscription the way PulseAudio does, so detecting external changes (like plugging in a monitor from another tool) relies on the 10-second polling tick.
- The arrangement view positions monitors by nudging in 10-pixel increments. For pixel-perfect placement, use the position dialog (`p` key).
- Mirror mode requires at least two monitors. Mirroring forces both outputs to the same resolution.
