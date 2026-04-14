# Appearance

The visual side of Hyprland: theme, fonts, gaps and borders, rounding, blur, shadows, animations, layout parameters, and cursor theme. Everything here edits your Hyprland config files (`~/.config/hypr/*.conf`) and tells Hyprland to reload so the change takes effect immediately — no logout, no restart.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Appearance page has six sub-sections in an inner sidebar:

1. **Theme** — the current omarchy theme, its accent color, background, and foreground. Switching themes runs the Omarchy theme-switcher under the hood.
2. **Fonts** — the active font family and size from your Hyprland config (and waybar / terminal if those read from the same source). Dialog-based picker with previews.
3. **Windows** — inner and outer gaps, border size, rounding radius, border color (accent vs inactive), active-window border gradient, decoration mode
4. **Effects** — blur enabled/size/passes, shadow enabled/range/color, animations enabled and curve
5. **Cursor** — cursor theme name, size, inherited from GTK or Hyprland-native
6. **Screensaver** — the tte-driven ASCII art screensaver that hypridle fires after 15 minutes of idle on a stock Omarchy install

Each sub-section renders the current values in detail rows and lets you edit them via action keys.

## How dark writes changes

Most settings route through `hyprctl keyword` so they apply to the running session without touching config files. The ones that *do* need a file write (theme, font) edit the right file atomically and then either signal Hyprland to reload or rebuild via the Omarchy theme tooling.

## Navigating — focus and the key flow

1. Sidebar has focus on landing. `j`/`k` picks a sub-section.
2. `enter` moves focus into the content region.
3. Action keys apply changes.
4. `esc` backs out.

## Actions — the complete keybinding reference

### Theme

- `t` — open the theme picker dialog. Shows every theme under `~/.local/share/omarchy/themes/`. Select and commit to apply the full theme (background, foreground, accent, border, status colors).

### Fonts

- `f` — open the font picker dialog. Lists font families detected on this system.
- `+` / `=` — increase the font size by 1pt (affects the Hyprland config's font size setting, which some bars and terminals read)
- `-` / `_` — decrease the font size by 1pt

### Windows (gaps, borders, rounding)

- `i` — gaps in + 1 (space between windows on the same workspace)
- `I` — gaps in − 1
- `o` — gaps out + 1 (space between windows and screen edges)
- `O` — gaps out − 1
- `r` — rounding + 1 (corner radius in pixels)
- `R` — rounding − 1
- `b` — cycle border size + 1 (thickness of the window border in pixels, capped at a sensible maximum)

### Effects (blur, shadow, animations)

- `B` — toggle blur globally (behind semi-transparent windows). Off is cheaper; on looks nicer.
- `z` — blur size + 1 (how far the blur kernel reaches)
- `Z` — blur size − 1
- `x` — blur passes + 1 (how many iterations of the blur kernel; more passes = smoother but slower)
- `X` — blur passes − 1
- `A` — toggle animations on/off globally

### Screensaver

- `e` — toggle the screensaver on or off. Creates or removes `~/.local/state/omarchy/toggles/screensaver-off`. When disabled, `omarchy-launch-screensaver` exits early and hypridle's 15-minute idle trigger is effectively a no-op.
- `c` — open the full-screen content editor pre-filled with `~/.config/omarchy/branding/screensaver.txt`. Edit the ASCII art with normal typing and arrow-key navigation, press `Ctrl+S` to save, or `Esc` to discard changes.
- `p` — run the screensaver live as a preview. Spawns `omarchy-launch-screensaver force`, which covers every monitor with the tte effect loop. **Any key exits** — not just `esc`. If the user never presses a key, a 60-second failsafe inside dark's service layer SIGTERMs the child so the TUI can never be permanently stuck.

### Universal

- `enter` — open a value dialog for rows where action keys don't apply
- `esc` — back out
- `?` — open this help drawer

## Dialogs

### Theme picker

Select list of every directory under `~/.local/share/omarchy/themes/`. Each one contains a palette, a background, and Hyprland / waybar / alacritty (or ghostty / kitty) theme files. Dark runs the omarchy theme-switcher tool on commit, which copies the theme files into place and signals every component to reload.

### Font picker

Dark enumerates fonts via `fc-list` and groups by family. The picker shows one entry per family — commit to write the family name into Hyprland's font key and every location that reads from it (waybar, the terminal config, etc., if you've set them to mirror).

Dialog controls:

- `j` / `k` — move selection
- `enter` — commit
- `esc` — cancel

### Screensaver content editor

Unlike the other dialogs in dark, the screensaver content editor is a full-screen overlay — ASCII art banners are typically wider than 48 columns, so a modal would be cramped. It wraps `bubbles/textarea`, so every navigation key you'd expect works:

- Normal typing and `backspace` edit the text
- Arrow keys and `home` / `end` move the cursor
- `Ctrl+Left` / `Ctrl+Right` jump words
- `Enter` inserts a newline (that's why submit isn't `Enter`)
- `Ctrl+S` commits the edit and fires the bus request to write the file
- `Esc` cancels without saving — no dirty-check prompt, so use it deliberately
- `Ctrl+C` still quits dark globally — the editor doesn't trap you

The write is atomic (write-to-tmp then rename) so a crash or kill mid-write can't leave you with a truncated or blank screensaver file.

## Common tasks

### Switch to a different Omarchy theme

1. Theme sub-section, press `t`.
2. Select from the list. `enter`.
3. Wait a second for the theme-switcher to apply. The desktop, bar, and terminal should all recolor in place.

### Make the font a little bigger

1. Fonts sub-section, press `+` a few times.
2. Each press increments by 1pt. Watch the current size on the Size row.
3. The change applies to Hyprland immediately; apps that read from the same font source pick it up on their next render.

### Tighten the window layout

1. Windows sub-section, focus the content.
2. Press `I` to shrink inner gaps (space between windows).
3. Press `O` to shrink outer gaps (space between windows and edges).
4. If your windows are too "boxy", press `R` to reduce rounding.

### Turn off blur and animations for performance

1. Effects sub-section.
2. Press `B` to toggle blur off. The Blur row flips to `disabled`.
3. Press `A` to toggle animations off. Windows snap instead of sliding.
4. Both changes persist via `hyprctl` — add matching lines to your config for them to persist across sessions.

### Give active windows a thicker border

1. Windows sub-section.
2. Press `b` to step through border sizes — typically 1 / 2 / 3 / 4 / 5 pixels.
3. Each press also updates the gradient definition so the accent color scales appropriately with the thicker border.

### Change the cursor theme and size

1. Cursor sub-section.
2. Press `enter` on the Theme row to open a picker (lists themes from `~/.icons/` and `/usr/share/icons/`).
3. Press `enter` on the Size row to set a pixel size (24 / 32 / 48 / 64 are standard).
4. Changes take effect when the cursor next enters a new window — in practice, as soon as you move the mouse.

### Preview the screensaver without waiting 15 minutes

1. Screensaver sub-section. Check the Dependencies box first — `tte` must be installed and your terminal must be one of alacritty / ghostty / kitty. If either is red or gold, preview won't work.
2. Press `enter` to focus the content region.
3. Press `p`. Dark fires `omarchy-launch-screensaver force` and the tte effect loop takes over every monitor.
4. **Press any key** (esc works, so does space or any letter) to exit. The terminal's `read -n1` picks up the byte and the screensaver's trap tears everything down.
5. If you somehow get stuck, the 60-second failsafe inside dark will SIGTERM the child and you'll be returned to the TUI automatically.

### Edit the screensaver ASCII art

1. Screensaver sub-section, `enter` to focus.
2. Press `c`. The full-screen editor opens with the current contents of `~/.config/omarchy/branding/screensaver.txt`.
3. Edit freely — tab width, cursor position, and typing all work as in a normal textarea.
4. Press `Ctrl+S` to save. Dark writes atomically and fires a new snapshot so the preview box updates right away.
5. Press `p` to see the new banner in action.

If you want to start from a blank canvas, select all (dark's textarea supports the usual emacs / readline keybinds — `Ctrl+A` start of line, `Ctrl+E` end of line, etc. — though select-all isn't mapped), delete, and type your art.

### Disable the screensaver without removing anything

1. Screensaver sub-section, `enter` to focus.
2. Press `e`. The State row flips from `enabled` to `disabled` and the kill-switch file is created.
3. `omarchy-launch-screensaver` will now exit silently when hypridle triggers it, so the screensaver never runs until you press `e` again.
4. The ASCII art file and the hypridle timeout stay exactly as they were — toggling enabled just gates the launcher.

## Persistence: live vs. config file

Dark's fast path is `hyprctl keyword …`, which only affects the running session. Restarting Hyprland loses the change. For persistent values, add them to `~/.config/hypr/hyprland.conf` (or a sourced file):

```
general {
  gaps_in = 5
  gaps_out = 10
  border_size = 2
  col.active_border = rgb(a6e3a1)
}
decoration {
  rounding = 8
  blur {
    enabled = true
    size = 6
    passes = 2
  }
  drop_shadow = true
}
animations {
  enabled = true
}
```

The theme and font dialogs DO write to config files (because the theme-switcher tooling requires that), so those persist automatically. The quick-toggle keys (gap/border/rounding/blur/animation) are session-only unless you hand-copy the values into your config.

## Data sources, for the curious

- **`hyprctl getoption <key>`** — live values for every Hyprland setting dark surfaces
- **`~/.local/share/omarchy/themes/`** — theme list for the picker
- **`fc-list`** — font family list for the font picker
- **`~/.icons/` and `/usr/share/icons/`** — cursor theme list
- **`~/.config/hypr/hyprland.conf`** (and sourced includes) — parsed to populate the font family row and cursor theme row
- **The omarchy theme-switcher binary** — called to apply a theme selection across every relevant config file

Screensaver-specific data sources:

- **`~/.local/state/omarchy/toggles/screensaver-off`** — kill-switch flag file. Presence means disabled.
- **`~/.config/omarchy/branding/screensaver.txt`** — ASCII art banner fed to `tte`.
- **`exec.LookPath("tte")`** — tte installed check (the python package `terminaltexteffects`).
- **`xdg-terminal-exec --print-id`** — current terminal detection. Mapped to alacritty / ghostty / kitty / foot / wezterm / other. Only the first three are supported by `omarchy-launch-screensaver`.
- **`omarchy-launch-screensaver force`** — invoked by the preview action; blocks until every screensaver window exits or the 60-second failsafe fires.

Dark publishes a fresh appearance snapshot on `dark.appearance.snapshot` after every write and on a periodic tick. The screensaver state publishes separately on `dark.screensaver.snapshot` because it's independent of the rest of the appearance data — dark only republishes it on command completion, not on a periodic tick, since the flag file and content file rarely change underneath.

## Known limitations

- Gap/border/rounding/blur/animation changes are session-only unless you persist them by hand in `hyprland.conf`.
- The theme picker requires the Omarchy theme tooling to be installed — on a stock Arch without Omarchy, switching themes won't work.
- Font size changes only apply to Hyprland's own font key. Your terminal or bar may have independent font sizes you'd need to edit separately.
- Border gradient colors are read but can't be edited from dark — you need to hand-edit the `col.active_border = ...` line.
- Shadow settings (`drop_shadow`, `shadow_range`, `shadow_render_power`, `col.shadow`) are displayed but not individually toggleable from dark yet.
- Custom animation curves aren't editable — dark has an on/off toggle and reads the active curve name, but you can't pick between `default`, `windows`, `linear`, etc. from the UI.

Screensaver-specific limitations:

- Dark edits the ASCII art and the enabled flag, but **not** the `tte` effect list, frame rate, terminal choice, or the `on-timeout` command hypridle invokes. Those all live in Omarchy shell scripts that would be rewritten on the next omarchy upgrade — patching them would break the upgrade path.
- The trigger timeout lives on Privacy → Screen Lock, not here. It's a hypridle listener block, same as the lock and DPMS timers, and moving just one would fragment the mental model.
- Preview requires one of the three terminals `omarchy-launch-screensaver` knows how to configure (alacritty, ghostty, kitty). foot / wezterm / other terminals will show a notification and the preview action will no-op.
- `tte` is a Python package (`terminaltexteffects`). Without it the launch script exits silently — dark surfaces the dependency state in the Dependencies box so the no-op is visible instead of confusing.
- There's no content library to pick from — you edit the file directly. Pasting in a pre-generated banner is fine; the textarea doesn't validate anything.
- The preview failsafe is 60 seconds. If you need a longer interactive session with the screensaver up, trigger it from a terminal instead of from dark.
