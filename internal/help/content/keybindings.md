# Keybindings

Browse and edit the Hyprland keybindings on this machine — both the defaults that Omarchy ships and the user overrides in `~/.config/hypr/bindings.conf`. Dark parses the live config, shows every binding in one scrollable table, and writes any edits back through `hyprctl reload` so they take effect without a logout.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Keybindings page is one of the F3 → Omarchy sub-tabs. The left column is an inner sidebar with three filter sections:

1. **All** — every binding dark has parsed, regardless of source
2. **Default** — only bindings from Omarchy's default config files
3. **User** — only bindings from your personal `~/.config/hypr/bindings.conf`

The content pane shows a single table with columns:

- **#** — the row number within the current filter, 1-based
- **Modifiers** — the mod chord (e.g. `SUPER`, `SUPER SHIFT`, `SUPER CONTROL ALT`). `—` for bindings with no modifier (typically hardware keys like volume).
- **Key** — the key name, with evdev codes and XF86 symbols mapped to friendly labels. `XF86AudioRaiseVolume` shows as `Vol Up`, `code:163` as `Next Track`, and so on. Regular letter keys show as themselves.
- **Description** — the human-readable description from the `# comment` preceding the `bind =` line, or the dispatcher name if no comment was found.
- **Source** — `default` for bindings dark picked up from Omarchy's shipped config, or `user` for anything in your own bindings file.

Below the table:

- A hint line with the available action keys (`enter edit · a add · d delete`)
- A summary with the total visible count and, if the table scrolls, the current viewport range (`42 bindings  ↕ 10-25/42`)

## How dark finds your bindings

The live state comes from parsing these files in order:

1. **Default bindings** — dark reads `~/.local/share/omarchy/default/hypr/hyprland.conf`, walks every `source = …` directive, and for each sourced file under `~/.local/share/omarchy/default/hypr/bindings/` parses out the `bind = …` / `bindd = …` lines. Bindings from these files get tagged `source=default`. If `hyprland.conf` can't be parsed, dark falls back to reading every `.conf` file in the default bindings directory directly.
2. **User bindings** — `~/.config/hypr/bindings.conf` is parsed next. Every binding here is tagged `source=user`.
3. **User unbinds** — the user file can also contain `unbind = MOD, KEY` lines, which dark applies as a filter: any default binding whose `(mods, key)` matches an `unbind` is dropped from the final list. This is how you "remove" a shipped Omarchy binding without editing the default config files.

The result is one merged list you see in the table. Adds, edits, and removes only ever touch `~/.config/hypr/bindings.conf` — dark never writes to the Omarchy default files. To override a shipped default, add a new user binding on the same `(mods, key)`; to hide a default entirely, delete the row (dark writes an `unbind` line for you).

After any write, dark runs `hyprctl reload` so Hyprland picks up the new config immediately. No logout or session restart needed.

## Navigating — focus and the key flow

1. On landing, focus is on the inner sidebar's **All / Default / User** filter list. `j` / `k` moves between filters and the table below updates live.
2. Press `enter` to move focus into the table. The table border lights up in accent blue.
3. `j` / `k` or `↑` / `↓` moves the row selection. The viewport auto-scrolls to keep the selection visible when the table is taller than the pane.
4. `esc` returns focus from the table back to the filter list. `esc` again returns to the F3 Omarchy-level sidebar.

Hardware keys (`XF86AudioRaiseVolume` et al.) are rendered as their friendly names for readability. The table sort is whatever order the parser encountered the bindings in — typically file order.

## Actions — the complete keybinding reference

### From the filter sidebar (before you've pressed enter)

- `j` / `k` or `↑` / `↓` — cycle the filter: All → Default → User → All
- `enter` — move focus into the bindings table
- `?` — open this help drawer
- `esc` — back out to the Omarchy-level sidebar

### From the bindings table

- `j` / `k` or `↑` / `↓` — move the row selection
- `a` — add a new binding. Opens a five-field dialog: Modifiers, Key, Description, Dispatcher, Arguments.
- `enter` — edit the selected binding. Opens the same dialog pre-filled with the current row's values.
- `d` — delete the selected binding. Opens a confirmation dialog. For a user binding, this removes the line from `bindings.conf`. For a default binding, dark writes an `unbind = …` entry to `bindings.conf` so the filter layer hides it.
- `esc` — return focus to the filter sidebar

### Universal

- `ctrl+c` — quit from anywhere, including inside a dialog
- `?` — open this help drawer
- `ctrl+r` — rebuild dark and hot-reload

## Dialogs

The add and edit dialogs both ask for five fields:

- **Modifiers** — space-separated mod chord. Standard values: `SUPER`, `SHIFT`, `CONTROL`, `ALT`. Combine like `SUPER SHIFT` or `SUPER CONTROL ALT`. Leave empty for hardware keys or one-key bindings.
- **Key** — the key identifier. Letters are lowercase (`a`, `return`, `space`). Hardware keys use XF86 symbols (`XF86AudioRaiseVolume`) or evdev codes (`code:163`). Dark will render them as friendly names in the table but stores them verbatim in the config file.
- **Description** — a short label for the binding. Written as a `# <desc>` comment line before the `bind = …` line. The table uses this for the Description column; leave empty to fall back to the dispatcher name.
- **Dispatcher** — the Hyprland dispatcher to invoke. Common values: `exec` (run a shell command), `exec-once`, `killactive`, `togglefloating`, `workspace`, `movetoworkspace`, `fullscreen`, `movefocus`, `resizeactive`, `pin`.
- **Arguments** — the argument string passed to the dispatcher. For `exec`, this is the command to run. For `workspace`, a workspace number. For `movefocus`, a direction (`l`, `r`, `u`, `d`).

Dialog controls:

- `tab` / `↓` — next field
- `shift+tab` / `↑` — previous field
- `backspace` — delete the previous character
- `enter` — submit
- `esc` — cancel

### Conflict detection

When you try to add or edit a binding onto a `(mods, key)` combination that already has a binding, dark catches the conflict before writing and pops a **Conflicts with: …** confirmation dialog. Choosing to confirm writes the new binding anyway — Hyprland will then use whichever line is parsed last — and fires a desktop notification listing the conflicts so you have a record outside the TUI. Choosing to cancel drops the change and leaves the existing bindings alone.

The detection is scoped to the full merged list (default + user), so overriding a default binding is explicit rather than silent.

## Common tasks — how to actually do things

### Find a binding

The table is source-filtered, not text-searched. To narrow down:

1. If you know a binding is user-defined, set the filter to **User**.
2. If you know it came from Omarchy defaults, set it to **Default**.
3. Otherwise use **All** and scroll — the viewport shows roughly 20–40 rows at a time depending on terminal height.

Per-row text search isn't wired up yet; it's on the roadmap.

### Override a default binding with your own command

1. Filter to **All** and find the default binding you want to change.
2. Press `enter` to edit. This does NOT modify the default file — it writes a new line to `bindings.conf` with the same `(mods, key)` and your chosen dispatcher/args, and Hyprland's last-wins semantics make yours take precedence.
3. The row now appears twice if you keep the filter on All (once as default, once as user). Switch to User to confirm your override.

If you want the default binding to disappear entirely rather than being overridden, use "Delete" instead — dark writes an `unbind` line and the default is filtered out.

### Delete a binding

1. Select the row.
2. Press `d`. Confirm the dialog.
3. For a **user** binding, dark removes the line from `bindings.conf`.
4. For a **default** binding, dark writes `unbind = <MODS>, <KEY>` to `bindings.conf`. The default binding is still in the Omarchy file but the unbind suppresses it. The row disappears from dark's table on the next reload.

Both cases run `hyprctl reload` so Hyprland drops the binding immediately.

### Add a new binding to launch an app

1. Press `a`.
2. Fill in the fields:
   - Modifiers: `SUPER`
   - Key: `t`
   - Description: `Launch terminal`
   - Dispatcher: `exec`
   - Arguments: `alacritty` (or whichever command you want)
3. Press `enter` to submit. If the `(SUPER, t)` combination already exists, you'll get the conflict dialog — confirm or cancel from there.
4. The binding appears in the User filter and becomes active immediately via `hyprctl reload`.

### Add a binding for a hardware key

Hardware keys don't have modifiers, so leave the Modifiers field empty.

1. Press `a`.
2. Modifiers: leave blank.
3. Key: `XF86AudioRaiseVolume` (or whatever the key identifies as — `wev` or `hyprctl monitors` can help you find the right name).
4. Dispatcher: `exec`.
5. Arguments: `wpctl set-volume @DEFAULT_AUDIO_SINK@ 5%+`.
6. Submit. The new binding will show in the User filter with the friendly name `Vol Up` in the Key column.

### Turn off a shipped Omarchy binding you never use

Use the Delete flow on a default binding — dark writes an `unbind` line that suppresses it without touching the default config files. This is the one-way form of override: you can always restore the binding by deleting the `unbind` entry (switch to User filter, find the `unbind` line, delete it).

## Data sources, for the curious

- **Default bindings directory** — `~/.local/share/omarchy/default/hypr/bindings/`. Dark walks the `source = …` directives in `~/.local/share/omarchy/default/hypr/hyprland.conf` to decide which files in this directory are active, then parses `bind = …` / `bindd = …` / `binde = …` / `bindl = …` / `bindm = …` / `bindn = …` / `bindo = …` / `bindr = …` / `bindt = …` lines from each.
- **User bindings file** — `~/.config/hypr/bindings.conf`. Created by dark on first add if it doesn't exist. Dark writes in a stable format (one blank line between bindings, `# description` comments preserved) so hand-editing between dark sessions is safe.
- **Friendly key-name map** — a built-in lookup table in `internal/tui/keybind_view.go` that maps the common XF86 symbols and evdev codes to readable labels. Unknown keys fall through to their raw name.
- **Hyprland reload** — `hyprctl reload` is invoked after every successful write. If `hyprctl` fails, the config file is still updated but the live session stays on the old bindings until the next reload (or logout/login).

Dark publishes a fresh keybind snapshot on `dark.keybind.snapshot` every time a write completes and on a periodic tick, so if you edit `bindings.conf` outside of dark the table catches up within a few seconds.

## Known limitations

- The table doesn't support text search yet — filter by source is the only cut.
- `submap` blocks (Hyprland's modal keybindings) aren't parsed. They still work in Hyprland but dark won't show them.
- The dispatcher list isn't validated against Hyprland's real set. You can type a nonsense dispatcher and dark will happily write it; `hyprctl reload` will print a warning but not stop the write.
- There's no "restore default" button for a binding you've unbound. You have to delete the `unbind` line in the User filter.
- Key-combo parsing is literal: `SUPER SHIFT` and `SHIFT SUPER` are treated as the same chord (dark sorts them alphabetically before comparing), but `SUPER+SHIFT` with a `+` separator will not match either.
