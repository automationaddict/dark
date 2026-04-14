# Notifications

Configure the desktop notification daemon (mako on a stock Omarchy install) — where notifications appear, how long they stay up, what they sound like, their width and visibility cap, and per-app rules for hiding or altering individual notifications. Every edit writes to `~/.config/mako/config` and tells mako to reload so changes take effect on the next notification.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The Notifications page has four sub-sections in an inner sidebar:

1. **Daemon** — which notification daemon is running (mako / dunst / gnome-shell / something else), its PID, and the path of the config file dark reads. If the daemon isn't mako, most of the controls go read-only because dark only knows how to rewrite mako's config shape.
2. **Appearance** — anchor position, window width, text style, layer, max visible count, icons on/off.
3. **Behavior** — per-urgency timeouts (low / normal / critical), default sound, DND state, a one-shot "dismiss all" action.
4. **Rules** — per-app rules that override appearance or behavior for a specific `app-name`. Hide notifications from a noisy app, extend the timeout for critical ones, or swap the sound.

Every row in every sub-section is live-editable by pressing an action key or `enter`.

## How dark reads and writes the daemon config

Dark parses `~/.config/mako/config` on first load and every time a write completes. The parser understands:

- Top-level `key=value` pairs in the global section
- `[urgency=low|normal|critical]` sections with their own timeout and sound overrides
- `[app-name=<pattern>]` sections with their overrides

Writes rewrite the whole file atomically and run `makoctl reload` (or the equivalent for your daemon if it's something other than mako). If `makoctl` isn't on the PATH, the write still happens but the live daemon keeps running the old config until it's restarted.

## Navigating — focus and the key flow

1. Sidebar has focus on landing. `j`/`k` picks a sub-section.
2. `enter` moves focus into the content region.
3. Action keys on the highlighted row apply changes.
4. `esc` backs out.

## Actions — the complete keybinding reference

### Appearance

- `p` — cycle the anchor position through seven positions: top-right → top-left → top-center → bottom-right → bottom-left → bottom-center → center → top-right
- `w` — increase notification width by 20px (max 800)
- `W` — decrease notification width by 20px (min 100)
- `l` — cycle the wayland-layer between `overlay` (above fullscreen apps) and `top` (normal layer)

### Behavior

- `d` — toggle Do Not Disturb. While DND is on, every notification is suppressed at the daemon level — mako stores them in its queue but doesn't display them. Turn DND off to see what queued up.
- `D` — dismiss all currently visible notifications (one-shot, not a setting)
- `+` / `=` — increase the timeout for the highlighted urgency by 1000ms (max 30s)
- `-` / `_` — decrease by 1000ms (min 1s)
- `o` — open the sound picker dialog. Navigate with `j`/`k` and the sound previews as you scroll. Press `enter` to commit or `esc` to cancel.
- `O` — disable the default sound entirely (equivalent to picking "no sound")

### Rules

- `a` — open the Add rule dialog. Two fields: App name (e.g. `Firefox`, `slack`, `Discord`) and Action (`hide` to suppress, `show` to force visible even during DND).
- `x` — open the Remove rule dialog. Type the full criteria string from the rule list to delete it.

### Universal

- `enter` — open a value edit dialog where applicable (otherwise acts as focus / apply)
- `esc` — back out
- `?` — open this help drawer

## Dialogs

### Sound picker

Dark walks `/usr/share/sounds/freedesktop/stereo/*.oga` to build the sound list. When you arrow through the list each focused sound auto-previews via `mpv --no-terminal --no-video <path>`, stopping the previous preview immediately so only one sound plays at a time. Press `enter` to commit — dark writes the path into the mako config's `default-sound` key. `esc` cancels and stops the preview.

### Add rule

Two fields: App name and Action. App name matches the notification's `app-name` attribute, which is what programs set via `NotifySend` or libnotify. You can find it by checking `makoctl list` after the app fires a notification. Action is `hide` (suppress) or `show` (force visible even while DND is on).

Dark writes a new `[app-name=<App>]` section with either `invisible=1` (hide) or `invisible=0` + `on-notify=none` (show).

### Remove rule

Confirmation dialog for deleting a named rule — type the criteria exactly as shown in the Rules list.

## Common tasks

### Move notifications to the top-left of the screen

1. Appearance sub-section, `p` to cycle anchor until it reads `top-left`.

### Quiet Slack without killing the whole daemon

1. Rules sub-section, `a` to add.
2. App name: `Slack`. Action: `hide`. `enter`.
3. Slack notifications go straight to DND-queue and never pop up. Remove the rule with `x` when you want them back.

### Extend the timeout for critical notifications so you don't miss them

1. Behavior sub-section, `enter` to focus the content.
2. Arrow to the `Critical timeout` row.
3. Press `+` several times to ratchet up to 15–30 seconds (0 means "stay forever until dismissed").

### Make notifications quieter / pick a different sound

1. Behavior sub-section.
2. `enter` to focus, arrow to the sound row.
3. `o` to open the picker. `j`/`k` auto-previews each one.
4. `enter` to commit the highlighted sound, or `O` from the row to disable sound entirely.

### Turn off Do Not Disturb and see what you missed

1. Behavior sub-section.
2. Press `d` to toggle DND off. Queued notifications from mako's stash appear in a burst, unless they've already expired.

### Clear out a cluttered notification stack

From anywhere on the page, press `D` (capital) to dismiss every visible notification. This is a one-shot action, not a setting — nothing persists.

## Data sources, for the curious

- **`~/.config/mako/config`** — parsed and rewritten directly
- **`makoctl list`** — live notification list (used when computing "dismiss all")
- **`makoctl reload`** — invoked after every config write
- **`makoctl mode <dnd|default>`** — toggles DND at the daemon level
- **`/usr/share/sounds/freedesktop/stereo/`** — the sound picker's source
- **`mpv --no-terminal --no-video <path>`** — sound preview player

Dark publishes fresh notify snapshots on `dark.notify.snapshot` after every config write and on a periodic tick, so hand-edits to the config file are picked up within a few seconds.

## Known limitations

- Only mako is a first-class target. If you're running dunst or another daemon, dark shows the state but edit keys may no-op.
- The sound picker only walks `/usr/share/sounds/freedesktop/stereo/`. System sounds in other directories (e.g. custom sounds under `~/.local/share/sounds/`) aren't surfaced yet.
- DND mode only has two states (on / off). More granular scheduling (e.g. "DND during working hours") needs a wrapper like `mako-dnd-scheduler` or a cron job; dark doesn't surface it.
- Per-urgency appearance overrides (different colors for critical vs normal) aren't editable in dark. You can still hand-edit `~/.config/mako/config` and dark will read but not round-trip the values.
- Rule matching is on `app-name` only. Matching on summary text or category requires hand-editing.
