# Workspaces

Browse every Hyprland workspace on this machine — which ones are active on which monitors, what they're tiling (dwindle vs master), and the handful of workspace-adjacent config knobs that shape how switching workspaces feels. Every mutation routes through `hyprctl` and applies to the running session immediately.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Workspaces page has three sub-sections in an inner sidebar:

1. **Overview** — the live workspace table. One row per workspace Hyprland has created, plus any persistent rules from your config. The active workspace is marked with a dot in the row marker and rendered in the accent color.
2. **Layout** — the global default layout (dwindle / master) plus per-layout options (dwindle's pseudotile / preserve_split / force_split / smart_split / smart_resizing and master's new_status / orientation).
3. **Behavior** — workspace-navigation feel knobs: cursor warp on workspace change, the global animations toggle, hide-special-on-workspace-change.

All data comes from `hyprctl workspaces -j`, `hyprctl workspacerules -j`, `hyprctl activeworkspace -j`, and `hyprctl getoption <key>` reads. A periodic 3-second tick refreshes the snapshot so opening the panel always shows current reality even if you've been switching workspaces from the keyboard.

## How dark writes changes

Every setter calls `hyprctl keyword <key> <value>` (for layout options and behavior flags) or `hyprctl dispatch <verb> <args>` (for workspace navigation, rename, and monitor moves). These changes apply to the **current session** only. To persist them across Hyprland restarts you still need to add the equivalent lines to `~/.config/hypr/hyprland.conf` or a sourced include — same caveat as the Appearance panel.

The Omarchy shell wrappers (`omarchy-hyprland-workspace-layout-toggle` etc.) are deliberately **not** invoked. Dark reimplements the logic in Go so the backend is directly testable and doesn't depend on those wrapper scripts being on PATH.

## Navigating — focus and the key flow

1. On landing, focus is on the inner sidebar's **Overview / Layout / Behavior** list. `j`/`k` moves between sub-sections and the content pane updates live.
2. Press `enter` to move focus into the content region. For Overview, the content is the workspace table and `j`/`k` moves the row selection. For Layout and Behavior, the content is a set of detail rows with per-row action keys.
3. `esc` returns focus from the content region to the sub-section sidebar, then from there to the main Settings sidebar.

## Actions — the complete keybinding reference

### Overview

Operates on whichever workspace row is highlighted.

- `j` / `k` — move the row selection
- `enter` — switch to the selected workspace (dispatches `hyprctl dispatch workspace <id>`). If the workspace doesn't exist yet, Hyprland creates it.
- `r` — open a rename dialog pre-filled with the workspace's current name. Submit writes `hyprctl dispatch renameworkspace <id> <name>`. Note that rename is session-only; on Hyprland restart the workspace reverts to its numeric name unless you add a persistent rule.
- `m` — open a select dialog listing every connected monitor and move the selected workspace there. Requires at least two monitors; a notification fires on single-monitor setups.
- `L` — cycle the per-workspace layout between dwindle and master. Dispatches `hyprctl keyword workspace <id>, layout:<layout>`.

### Layout

Operates on the global default layout and the dwindle / master option blocks.

- `L` — cycle the global default layout (dwindle ↔ master). New workspaces pick up the new default; existing workspaces keep their current per-workspace layout.
- `t` — toggle dwindle `pseudotile`. When enabled, single windows on a workspace float at their preferred size instead of filling the container.
- `p` — toggle dwindle `preserve_split`. Controls whether the split direction persists when you move windows around.
- `f` — cycle dwindle `force_split` through `auto` (0), `left/top` (1), `right/bottom` (2). Determines which side new windows land on when they split a container.
- `M` — cycle master `new_status` through `master` / `slave` / `inherit`. Controls whether a newly-opened window becomes the master, a slave, or inherits from its parent's position.

Smart split and smart resizing are read-only in dark for now — the snapshot shows their current state but there's no toggle key because they're edge-case options most users never touch.

### Behavior

- `c` — toggle `cursor:warp_on_change_workspace`. When enabled, switching workspaces moves the mouse cursor to the new focus target. When disabled, the cursor stays where it is.
- `a` — toggle `animations:enabled`. This is the global Hyprland animations flag and gates every animation, not just workspace transitions. If you only want to kill the workspace fade specifically, edit `animations { animation = workspaces, 0 ... }` in your config directly.
- `h` — toggle `binds:hide_special_on_workspace_change`. When enabled, the special workspace (Hyprland's hidden-by-default overlay) auto-hides when you switch to a regular workspace. When disabled, it stays visible until you explicitly dismiss it.

### Universal

- `?` — open this help drawer
- `esc` — back out one level
- `ctrl+c` — quit from anywhere
- `ctrl+r` — rebuild dark in place (dev workflow)

## Dialogs

### Rename workspace

Single field prefilled with the current name. Submit writes
`hyprctl dispatch renameworkspace <id> <name>` and the Overview
table updates on the next snapshot tick (within 3 seconds).

### Move workspace to monitor

Select field listing every connected monitor from the display
snapshot dark already has. Requires the Display service to have
loaded at least once — if you opened dark straight to Workspaces
before Display ticked, you may see an empty list; wait a second
and retry.

## Common tasks

### Jump to a specific workspace from the TUI

1. F1 → Workspaces → Overview → `enter` to focus the table.
2. Arrow to the workspace you want.
3. `enter`. Hyprland switches to that workspace instantly. Dark stays visible on the new workspace as long as it's not a fullscreen tiling overlap.

### Move a workspace to the other monitor

On a laptop + external display setup:

1. Overview → `enter` to focus the table.
2. Arrow to the workspace that's currently on the wrong monitor.
3. Press `m`. The monitor picker opens.
4. Select the target monitor. `enter`.
5. The workspace and all its windows move to the new monitor.

### Switch a single workspace to master layout

If you want one workspace to use master-stack (e.g. for focused coding) while everything else stays on dwindle:

1. Overview → `enter` → arrow to the workspace.
2. Press `L`. The per-workspace layout cycles to master.
3. Your other workspaces are unaffected.

### Set the global default layout to master

1. Navigate to Layout sub-section.
2. Press `L`. The Default Layout row flips from `dwindle` to `master`.
3. New workspaces created after this point use master. Existing workspaces keep whatever they had.

### Stop the cursor from following workspace switches

Helpful when you're using multiple monitors and don't want the cursor to jump every time you change workspaces:

1. Navigate to Behavior sub-section.
2. Press `c`. The Cursor Warp row flips from enabled to disabled.
3. The change applies immediately. Your cursor will no longer teleport to the focused window of the new workspace.

### Disable all animations for maximum snap

1. Behavior sub-section.
2. Press `a`. Animations row flips to disabled.
3. Workspace transitions, window opens, layer animations, everything — all instant from now on. Toggle back with another `a`.

## Data sources, for the curious

- **`hyprctl workspaces -j`** — live list. One workspace per row, with ID, name, monitor, monitor ID, window count, fullscreen flag, last-focused window, persistence flag, and tiled layout.
- **`hyprctl workspacerules -j`** — persistent workspace rules from `workspace = …` config directives. Surfaced in the Overview summary line so you know how many are active.
- **`hyprctl activeworkspace -j`** — current active workspace ID, used to paint the accent marker in the Overview row gutter.
- **`hyprctl getoption <key>`** — typed reads for `general:layout`, `dwindle:*`, `master:*`, `cursor:warp_on_change_workspace`, `animations:enabled`, `binds:hide_special_on_workspace_change`. Each read is one process, so the snapshot walks ~12 options in sequence — still sub-50ms on normal hardware.
- **`hyprctl dispatch …`** — workspace navigation, rename, and monitor moves.
- **`hyprctl keyword …`** — every layout and behavior setter.

Dark publishes a fresh workspaces snapshot on `dark.workspaces.snapshot` every 3 seconds (faster than most services because workspaces change every time the user switches) and on every action completion so the TUI's cached state stays in sync.

## Known limitations

- All changes are session-only unless you copy them into `hyprland.conf`. Dark doesn't round-trip writes into config files for workspace options yet.
- Persistent workspace rules (the ones from `workspace = …` config lines) are **read-only** in dark. The Overview summary shows how many exist but you can't add, edit, or remove them from the TUI — edit `hyprland.conf` directly for that.
- `special:` workspaces (Hyprland's overlay workspaces) are visible in the live list but treating them like regular workspaces in Overview can get confusing. Actions work on them but the UX is geared for numeric workspaces.
- The master-layout `orientation` option (left / right / top / bottom / center) is read-only from this panel. Use the raw hyprland.conf if you need to pick one.
- Smart split and smart resizing are displayed but have no toggle key — they're edge-case dwindle options most users never touch.
- The `workspaces` animation line under `animations { ... }` is read-only. Toggle the global animations flag with `a` on the Behavior sub-section, or edit the line directly for per-animation control.
- Dark doesn't wrap `movetoworkspacesilent` (the "move a window without following it" dispatcher) — that's a window-focused action rather than a workspace-focused one, so it belongs in a future Windows panel.
