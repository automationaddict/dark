# Links

Manage the three lists of launchable shortcuts dark maintains for the F3 → Omarchy → Links tab: **Web Links** (URLs that open in an Omarchy webapp wrapper), **TUI Links** (terminal programs that open in a pre-configured floating or tiled terminal window), and **Help Links** (documentation URLs that open in the default browser). Every edit writes to one YAML file and takes effect on the next `enter` press.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Links page is one of the F3 → Omarchy sub-tabs. The left column is an inner sidebar with three entries:

1. **Web Links** — URLs launched via `omarchy-launch-webapp`, which wraps them in a bordered Chromium window tagged as a desktop app
2. **TUI Links** — terminal programs launched via `xdg-terminal-exec` with a `TUI.float` or `TUI.tile` app-id so Hyprland can apply a pre-configured window rule
3. **Help Links** — documentation URLs that open in the default browser via the same webapp launcher path

Each list renders as a plain numbered table. The content pane shows:

- **Web Links**: `#`, `Name`, `URL`
- **TUI Links**: `#`, `Name`, `Command`, `Style` (where Style is `float` or `tile`)
- **Help Links**: same shape as Web Links

Below the table is a hint line with the action keys (`enter open · a add · e edit · d delete`).

## Why these three lists are separate

All three ultimately launch something, but their launch mechanics differ enough that merging them would blur the UX:

- **Web links** need the Omarchy webapp wrapper so the browser window opens as a real desktop-class app (its own class, its own taskbar entry, no tab chrome). That wrapper takes a URL and produces a webapp — you don't hand-craft a command.
- **TUI links** need a terminal emulator and a window-rule class so Hyprland can decide floating vs tiled placement. The command is whatever terminal program you want to run — `btop`, `helix`, `yazi`, `lazygit`, etc. — and dark wraps it in an `xdg-terminal-exec --app-id=TUI.<style>` invocation.
- **Help links** open in the default browser (no webapp wrapper), because they're meant for one-off reference material, not persistent apps.

Keeping them in three tables means the Add dialog can ask for exactly the right fields: URLs for web/help, Command+Style for TUI.

## Navigating — focus and the key flow

1. On landing, focus is on the inner sidebar's **Web / TUI / Help** list. `j` / `k` moves between sub-sections and the table below updates live.
2. Press `enter` to move focus into the table. The table border lights up in accent blue.
3. `j` / `k` or `↑` / `↓` moves the row selection.
4. `enter` inside the table **launches** the highlighted link (opens the webapp, starts the TUI terminal, or browses the help URL). It does NOT edit — use `e` for that.
5. `esc` returns focus from the table back to the sidebar. `esc` again returns to the F3 Omarchy-level sidebar.

## Actions — the complete keybinding reference

### From the inner sidebar (before you've pressed enter)

- `j` / `k` or `↑` / `↓` — cycle between Web Links, TUI Links, Help Links
- `enter` — move focus into the table
- `a` — add a new entry to the currently displayed list (works without entering the table — adds to whichever sub-section you're on)
- `?` — open this help drawer
- `esc` — back out to the Omarchy-level sidebar

### From the table

- `j` / `k` or `↑` / `↓` — move the row selection
- `enter` — **launch** the selected entry
- `a` — add a new entry
- `e` — edit the selected entry (opens the same dialog pre-filled)
- `d` — delete the selected entry (opens confirmation)
- `esc` — return focus to the inner sidebar

### Universal

- `ctrl+c` — quit from anywhere, including inside a dialog
- `?` — open this help drawer
- `ctrl+r` — rebuild dark and hot-reload

## Dialogs

### Web Link add/edit

Two fields:

- **Name** — the label that appears in the table and in the webapp's window class
- **URL** — the URL the webapp opens

Launch mechanic: `omarchy-launch-webapp <URL>`. Omarchy wraps this in a Chromium `--app` invocation so the resulting window gets its own class and can be targeted by Hyprland window rules.

### TUI Link add/edit

Three fields:

- **Name** — the label that appears in the table
- **Command** — the shell command to run inside the terminal (e.g. `btop`, `lazygit`, `helix ~/notes/todo.md`)
- **Style** — `float` or `tile`. Defaults to `float` if you leave it blank or type something else.

Launch mechanic: `xdg-terminal-exec --app-id=TUI.<style> -e <command>`. The app-id tells Hyprland's window rules to apply either the `TUI.float` or `TUI.tile` configuration — you set those up in your Hyprland config separately, and dark just picks which one to use.

### Help Link add/edit

Two fields:

- **Name** — the label that appears in the table
- **URL** — the documentation URL

Launch mechanic: same as Web Links — `omarchy-launch-webapp <URL>`. The distinction between Web and Help is purely organizational: Web Links are apps you use repeatedly, Help Links are docs you look up.

### Delete

Deletion uses a zero-field confirmation dialog: `Remove <name>?`. Press `enter` to confirm or `esc` to cancel.

### Dialog controls (all types)

- `tab` / `↓` — next field
- `shift+tab` / `↑` — previous field
- `backspace` — delete the previous character
- `enter` — submit
- `esc` — cancel

## How dark stores the lists

Everything lives in a single YAML file: **`~/.config/dark/links.yaml`**. The structure is three top-level arrays (`web_links`, `tui_links`, `help_links`), each with a list of entries. On first load dark tries:

1. Read `~/.config/dark/links.yaml` if it exists. This is the source of truth.
2. If the YAML file doesn't exist, dark imports from your `~/.local/share/applications/*.desktop` files. It scans every `.desktop` entry and:
   - If the `Exec=` line contains `omarchy-launch-webapp` or `omarchy-webapp-handler`, dark pulls out the URL argument and creates a Web Link entry.
   - If the `Exec=` line matches `xdg-terminal-exec --app-id=TUI.<style>`, dark pulls out the command and style and creates a TUI Link entry.
   - Everything else is ignored.
3. After the import, dark writes `links.yaml` for the first time so subsequent loads don't re-scan.

This means the very first time you open the F3 → Links tab, you might already see a bunch of pre-populated entries — those are the Omarchy-shipped webapps and TUI helpers that landed as `.desktop` files during install. Editing them through dark writes to `links.yaml` only; the `.desktop` files underneath are untouched.

### What if I add a .desktop file later?

Dark doesn't re-import `.desktop` files after `links.yaml` exists. If you install a new Omarchy webapp and want it to show up here, either add it through `a` in dark (recommended) or delete `~/.config/dark/links.yaml` to trigger a fresh re-import on the next load (you'll lose any dark-only additions).

## Common tasks — how to actually do things

### Launch a web app I've already set up

1. F3 → Links → Web Links (sidebar auto-selects the first sub-section).
2. `enter` to focus the table.
3. Arrow to the app you want.
4. `enter` again to launch. The Chromium webapp opens in a new window.

The same flow works for TUI links (arrow to the one you want, `enter` launches the terminal with the command inside it) and Help links (arrow to the one you want, `enter` opens the browser).

### Add a new web app

1. `j`/`k` to **Web Links** in the inner sidebar.
2. Press `a`. The **Add web link** dialog opens.
3. Type a Name (e.g. `GitHub`).
4. `tab` to URL, type the URL (`https://github.com`).
5. `enter` to commit.
6. The new row appears at the bottom of the table and is immediately launchable.

### Add a TUI shortcut for a terminal app

1. `j`/`k` to **TUI Links**.
2. Press `a`.
3. Name: `Resource monitor`. Command: `btop`. Style: `tile`.
4. `enter` to commit.
5. The row appears. Launching it runs `xdg-terminal-exec --app-id=TUI.tile -e btop` — so Hyprland's `TUI.tile` window rule applies (if you've set one up) and the terminal is tiled into the current workspace.

### Fix a typo in an existing entry

1. Navigate to the table. Focus the row.
2. Press `e`. The dialog opens pre-filled with the current values.
3. Edit the field. `enter` to commit. Dark removes the old row and writes the new one — if you renamed it, the position in the table may change.

### Remove an entry

1. Focus the row.
2. Press `d`.
3. Confirm the dialog with `enter`.
4. The row disappears. Dark saves `links.yaml` immediately.

### Use a custom launcher command instead of the Omarchy webapp wrapper

Not directly supported — the Web Links launcher always invokes `omarchy-launch-webapp`. If you want a different launch command, use a TUI Link with whatever command you need (`exec google-chrome-stable --app=...` etc.). The app-id on TUI links targets Hyprland terminal window rules, but if you set the command to a GUI program it will still run; it just won't match those rules.

## Data sources, for the curious

- **YAML store**: `~/.config/dark/links.yaml` — three arrays, written atomically on every add/edit/remove.
- **Desktop import fallback**: `~/.local/share/applications/*.desktop` — parsed once when `links.yaml` is missing, to seed the initial lists from Omarchy-installed webapps and TUI helpers.
- **Web launcher**: `omarchy-launch-webapp <URL>` — the Omarchy wrapper script that sets up the webapp window class.
- **TUI launcher**: `xdg-terminal-exec --app-id=TUI.<style> -e <command>` — the standard XDG terminal launcher with an app-id that matches Hyprland window rules you maintain separately.

Dark publishes a fresh links snapshot on `dark.links.snapshot` every time a write completes, so the table stays consistent even if you pop into another view and come back.

## Known limitations

- Adding a new entry always appends to the end of the list. There's no reordering — if you want alphabetical, you have to delete and re-add in order, or hand-edit `links.yaml`.
- The launch command is fixed per sub-section (webapp for Web/Help, xdg-terminal-exec for TUI). Custom launchers aren't supported.
- If you manually delete a `.desktop` file for a webapp Omarchy shipped, the entry remains in `links.yaml` because that file is now the source of truth. Delete it through dark with `d` to clean up.
- Launch errors from the wrapper scripts aren't surfaced in the UI — dark just fires the exec and moves on. If a webapp fails to open, check `journalctl --user -t omarchy-launch-webapp`.
- Help Links don't distinguish from Web Links in the launch mechanic; the two tables exist purely as organizational buckets.
