# Input Devices

Configure the keyboard, mouse, touchpad, and any other libinput-managed devices attached to this machine. Changes write to your Hyprland config via `hyprctl keyword input:…` calls, take effect immediately, and persist across sessions if you add the equivalent lines to your config files.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Input Devices page has four sub-sections in an inner sidebar:

1. **Keyboard** — layout, variant, options, repeat rate, repeat delay, numlock state
2. **Mouse** — acceleration profile, sensitivity, scroll method, natural scroll, middle-button emulation
3. **Touchpad** — sensitivity, tap-to-click, natural scroll, clickfinger vs software buttons, disable-while-typing, drag lock, scroll speed
4. **Other** — any other libinput devices dark detected (pen tablets, trackpoints, joysticks, etc.) with their raw device names and current bind state

Every row shows the current value read from `hyprctl devices` or from your Hyprland input config. Edits go through `hyprctl keyword` so they apply to the live session without a reload.

The keyboard layout list is populated from `/usr/share/X11/xkb/rules/evdev.lst` (if present) — dark parses that file once at startup to build the layout / variant lookup.

## Navigating — focus and the key flow

1. On landing, the inner sidebar has focus. `j`/`k` picks a sub-section.
2. Press `enter` to move focus into the content region.
3. Inside the content, `j`/`k` moves row selection.
4. Action keys on the highlighted row apply per-row changes (toggles, cycles, dialogs).
5. `esc` returns focus to the sidebar.

## Actions — the complete keybinding reference

### Keyboard

- `L` — open the keyboard layout dialog. Three fields: Layout (e.g. `us`, `de`, `gb`), Variant (e.g. `colemak`, `dvorak`, blank for none), Options (comma-separated XKB options like `ctrl:nocaps,grp:alt_shift_toggle`).
- `+` / `=` — increase keyboard repeat rate by 5 (characters per second once held)
- `-` / `_` — decrease keyboard repeat rate by 5
- `[` — decrease repeat delay by 50ms (time before repeat kicks in)
- `]` — increase repeat delay by 50ms

### Mouse

- `S` — decrease sensitivity by 0.05 (range is roughly -1.0 to 1.0; libinput treats negative values as deceleration)
- `s` — increase sensitivity by 0.05
- `a` — cycle the acceleration profile: `flat` / `adaptive` / (leave unset, i.e. libinput default)

### Touchpad

- `S` / `s` — sensitivity down / up (same scale as mouse)
- `t` — toggle tap-to-click (single-finger tap becomes a left-click)
- `n` — toggle natural scrolling (macOS-style inverted direction)
- `a` — cycle the acceleration profile (flat / adaptive / default)

### Universal

- `enter` — open a value dialog for the highlighted row when the action keys above don't apply
- `esc` — back out
- `?` — open this help drawer

## Dialogs

### Keyboard layout

Three fields — all optional except Layout. The typical combinations:

- `us` with no variant: standard US QWERTY
- `us` + variant `colemak`: Colemak on top of the US layout
- `de` + variant `neo`: German Neo layout
- `us,de` + option `grp:alt_shift_toggle`: dual-layout with `Alt+Shift` to switch

Dark validates none of this — whatever you type is passed to `hyprctl keyword input:kb_layout`, `input:kb_variant`, and `input:kb_options` verbatim. A bad value logs a warning in the Hyprland session but won't break your keyboard.

## Common tasks

### Set Caps Lock to Ctrl

1. Navigate to Keyboard.
2. Press `L` to open the layout dialog.
3. Leave Layout and Variant as-is. In Options, type `ctrl:nocaps`.
4. Press `enter`. The change applies immediately — Caps Lock now acts as Control.

### Switch to a Colemak layout

1. Press `L`.
2. Layout: `us`. Variant: `colemak`. Press `enter`.

### Make the trackpad less twitchy

1. Navigate to Touchpad.
2. Press `enter` to focus the content.
3. Press `S` a few times to lower sensitivity. Watch the current value on the Sensitivity row.
4. If the motion feels too flat, press `a` to cycle between flat and adaptive acceleration.

### Enable tap-to-click and natural scrolling for macOS feel

1. Touchpad sub-section.
2. Press `t` to enable tap-to-click.
3. Press `n` to enable natural scrolling.
4. Both changes are reflected in the rows and in Hyprland's live state.

### Speed up key repeat for gaming / vim

1. Keyboard sub-section.
2. Press `]` a couple times to shrink the repeat delay (try 200–300ms).
3. Press `+` a few times to increase repeat rate (try 30–50 cps).

## Persistence: live vs. config file

Dark writes every change via `hyprctl keyword …`, which only affects the **current session**. The next Hyprland launch reads from your config files and reverts to whatever's there. To make a change permanent, add the equivalent line to your `~/.config/hypr/hyprland.conf` (or a sourced file under `input { … }`):

```
input {
  kb_layout = us
  kb_variant = colemak
  kb_options = ctrl:nocaps
  repeat_rate = 40
  repeat_delay = 250
}
device {
  name = your-touchpad-name-here
  natural_scroll = true
  tap-to-click = true
}
```

Getting this wired into dark automatically — so changes persist without hand-editing — is a roadmap item.

## Data sources, for the curious

- **`hyprctl devices`** — live device list, current input config, per-device overrides
- **`/usr/share/X11/xkb/rules/evdev.lst`** — XKB layout/variant metadata (layouts, variants, options)
- **Hyprland's `input` D-Bus / socket interface** — for keyboard layout cycling and live reads

Dark publishes a fresh input snapshot on `dark.input.snapshot` every tick and after every write, so the displayed state stays in sync with what Hyprland thinks.

## Known limitations

- Changes only apply to the current Hyprland session. Persist them by hand-editing your config files (see above).
- Per-device configuration for multiple identical devices (e.g. two external mice) isn't surfaced — dark writes global `input { … }` settings, not per-`device { }` blocks.
- The Other sub-section lists unrecognized devices but doesn't provide configuration for them yet. You can see what Hyprland has detected, but editing is gated on known device classes.
- Pen / tablet / joystick devices aren't categorized into their own sub-sections; they all show under Other.
- Wayland-only: X11-specific input settings (via `xinput` / `xset r`) aren't touched because the target is Hyprland.
