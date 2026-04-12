# Sound

Manage the host's audio devices: see which outputs and inputs PipeWire knows about, check the default, adjust volume, mute, and switch the default output or input.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

Dark talks to the audio server over the PulseAudio native protocol. On Omarchy that server is PipeWire (via its `pipewire-pulse` compatibility shim). Everything on this page comes from `darkd`'s long-lived protocol client — no subprocess shell-out to `pactl`.

The page has up to five group boxes stacked vertically:

1. **Output Devices** — speakers, headphones, HDMI sinks, USB DACs, bluetooth headsets currently set to A2DP profile.
2. **Input Devices** — built-in microphone, USB microphones, bluetooth headsets currently set to a profile with a mic (HFP / HSP).
3. **Playing Applications** — every application currently playing audio into a sink (a "sink input" in PulseAudio terms). Only renders when at least one stream is active.
4. **Recording Applications** — every application currently capturing audio from a source (a "source output"). Only renders when at least one stream is active.
5. **Card** — reflects the highlighted device's backing card when focus is on Output Devices or Input Devices. Shows the active profile, the full list of available profiles, and the active port plus available ports. Hidden when focus is on an apps sub-table or when the highlighted device is a virtual sink with no card.

## Reading a row

Each row represents one device. Columns:

- **Name** — the human-readable device description (`Built-in Audio Analog Stereo`, `Sony WH-1000XM5`, etc.). Dark prefers the server's Description over the internal device name.
- **State** — `running` when audio is actively flowing, `idle` when the device is open but nothing is playing, `suspended` when the stack has powered it down to save energy. `suspended` is normal for an idle laptop — PipeWire re-wakes the device on demand.
- **Mute** — 󰕾 unmuted / 󰝟 muted.
- **Volume** — the per-channel average as a percentage, followed by a 10-cell bar graph. Volumes above 100% enter software over-amplification and can clip — the bar caps visually at full even if the number keeps rising.

Row markers:

- `▸` (accent) marks the selection cursor when this sub-table has focus.
- `★` (accent) marks the current **default** device. Applications that don't explicitly pick an output or input use the default.

## Navigating

1. Press `enter` to move focus into the content region. The Output Devices box border lights up in accent blue.
2. Press `tab` to cycle focus between the **Output Devices** and **Input Devices** sub-tables.
3. `j` / `k` (or arrow keys) moves the selection within the focused sub-table.
4. `esc` backs out to the sidebar.

## Actions

From sound content focus:

- `tab` — cycle focus through Output Devices → Input Devices → Playing Applications → Recording Applications, skipping any sub-table that's currently empty
- `j` / `k` — move the selection within the focused list
- `enter` — drill into the device info panel (Output / Input focus only)
- `+` / `=` — raise volume by 5 percentage points (up to 150% max). Operates on the highlighted device or app stream depending on focus.
- `-` / `_` — lower volume by 5 percentage points (minimum 0). Same focus-aware routing.
- `m` — toggle mute on the highlighted device or stream
- `p` — cycle to the next available profile on the highlighted device's card (Output / Input focus only)
- `o` — cycle to the next available port on the highlighted device (Output / Input focus only)
- `M` (shift+m) — move the highlighted app stream to the next available device of the matching kind. For a sink input, advances through the sink list; for a source output, advances through the source list. Wraps around. (Apps focus only.)
- `K` (shift+k) — kill the highlighted app stream. Disconnects it from PulseAudio entirely. Useful when an app refuses to release a sink. (Apps focus only.)
- `Z` (shift+z) — toggle suspend on the highlighted device. Suspending tells the audio server to release the underlying hardware so another client can grab it (or so the device can power down to save energy). The state column flips to `suspended`; press `Z` again to resume. (Output / Input focus only.)
- `D` (shift+d) — make the highlighted device the default (Output / Input focus only)
- `esc` — return focus to the sidebar (or close the device info panel)

## The Device Info panel

Press `enter` on a highlighted device to expand it. The Output/Input/Card boxes are replaced by a single device-info panel showing:

- **Description / Internal Name / Index** — full identifiers from the server.
- **State / Mute / Default / Channels / Active Port** — live state flags.
- A wide volume slider you can adjust with `+`/`-` and mute with `m`.
- Card header (name, driver, active profile) and profile list with `★` on the active profile and dimmed unavailable profiles.
- Port list with `★` on the active port and dimmed unplugged ports.

Action keys work on the same selected device while the panel is open. Press `esc` to back out to the device list.

## Why sinks and sources, not "speakers" and "mics"

PulseAudio's protocol uses *sink* for an output device (audio flows **into** it) and *source* for an input device (audio flows **out of** it). Dark relabels those as "Output Devices" and "Input Devices" in the UI because sink/source is confusing to anyone who hasn't worked with the stack. The terminology shows up in the code and in error messages from the daemon, but never in the primary UI.

## Cards, profiles, and ports

The Card box updates as you move the cursor through Output Devices and Input Devices. It shows three things:

- **Header** — the card's internal name, driver, and currently active profile.
- **Profiles** — every operating mode the card can be switched into. The active one is marked with `★`. Profiles the server reports as unavailable (e.g. bluetooth HFP when the remote end has it disabled) are dimmed and tagged `(unavailable)`.
- **Ports** — physical jacks/routes on the highlighted sink or source. The active one is marked with `★`. Unplugged jacks (the server reports `Available = 1`) are dimmed and tagged `(unplugged)`.

Press `p` to cycle to the next available profile on the highlighted device's card. The cycle skips unavailable profiles. Press `o` to cycle to the next available port on the highlighted sink/source.

## Bluetooth audio

Bluetooth headsets are card-backed devices. Switching between music quality and call-with-mic is a profile switch:

- **A2DP profiles** (`a2dp-sink`, `a2dp-sink-sbc_xq`, `a2dp-sink-aac`, `a2dp-sink-ldac`, etc.) expose a high-quality stereo output sink. No microphone.
- **Headset profiles** (`headset-head-unit`, `headset-head-unit-msbc`) expose a mono output sink and a mono input source. Lower quality but supports the microphone for calls.
- **Off** disconnects the audio profile entirely.

To switch your bluetooth headset between music and call modes:

1. Highlight the headset in Output Devices.
2. Press `p` to cycle to the next profile.
3. Repeat until the Card box shows the profile you want active.

When you switch from A2DP to a headset profile, the existing A2DP sink disappears and a new mono sink and mono source pair appear. Dark's snapshot picks them up on the next tick (or immediately after the action reply).

If the headset only exposes one available profile, `p` reports "no other available profiles on this card" — usually the remote end has the other profiles disabled or the codec negotiation hasn't completed yet.

## Per-application audio streams

The Playing Applications and Recording Applications boxes show every app currently producing or consuming audio. Each row covers one stream and lists:

- **Application** — the app name from the stream's PropList (`application.name`), combined with its current media name (track title, etc.) when both are present. Falls back to whichever is set, then to `(unnamed stream)`.
- **Routed To** — the description of the sink (for playback) or source (for recording) the stream is currently flowing through.
- **Mute** — `󰕾` unmuted / `󰝟` muted.

Each stream gets the same horizontal slider beneath its row as devices do, so per-app volume is right there. Volume and mute keys (`+`/`-`/`m`) operate on whichever sub-table currently has focus, so once you tab into Playing Applications you can adjust Spotify's volume independently from the system-wide sink volume.

### Move a stream between devices

1. Press `tab` until the cursor is on the Playing Applications box.
2. Arrow to the stream you want to move.
3. Press `M` (shift+m). The stream advances to the next available sink in the snapshot's sink list, wrapping around. Repeat until the Routed To column shows the device you want.

The same pattern works for Recording Applications, cycling between sources instead of sinks.

`M` reports an inline error when there's only one device available (nothing to move to). For two devices, two presses cycle back. For three+ devices, you may want to press `M` repeatedly to step through.

### Why per-app volume matters for bluetooth

The most common use case: you're playing music on speakers, want to take a call on your bluetooth headset, but don't want to disturb anyone listening to the speakers. Switch your headset to the headset-head-unit profile (Card box, `p` key) — that creates a new sink/source pair. Pull the call app out of Output Devices and into the headset by tabbing into Playing Applications, highlighting the call stream, and pressing `M`. Music keeps playing on the speakers because the music app's stream is still routed there.

## Headphone vs speaker switching

On a laptop with a single integrated audio card, headphones and speakers are usually two ports on the same sink rather than two separate sinks. Highlight the laptop's output device and press `o` to cycle to the next available port. Plugging in headphones flips the headphones jack from `(unplugged)` to available — most cards then route audio there automatically, but `o` lets you force the choice.

## Data source

Everything on this page comes from `darkd`, which keeps a long-lived PulseAudio protocol client open against the session socket at `/run/user/<uid>/pulse/native`. Commands issue a request and the server replies with a fresh state snapshot dark publishes on `dark.audio.devices`.

The daemon also subscribes to PulseAudio's property change events at startup, so any sink/source/card change another tool makes (pavucontrol, wpctl, Plasma sound widget, an app connecting to the server) is reflected within ~75 milliseconds — no polling delay. A safety-net snapshot publish happens every 30 seconds in case an event is ever missed.

## Live VU meters

The small indicator at the right end of each volume slider is a center-anchored stereo VU meter. Silence sits in the middle; the left channel grows leftward from center, the right channel grows rightward. Cells light up in green as the level rises and hit gold then red at the outer edges when the signal approaches clipping.

The daemon opens a peak-detect record stream against every sink's monitor source and every input source, configured for two channels. Mono sources are upmixed by the server (PulseAudio duplicates the channel when we ask for stereo against a mono input), so a built-in mic produces a perfectly symmetric meter. The latest readings are published on `dark.audio.levels` at 20 Hz. Muted devices show all-dim cells. Stream rows (Playing / Recording Applications) display the meter of the sink or source they're routed through, since per-app peak detection isn't part of the PulseAudio protocol.

Wave 1 covered sinks, sources, defaults, volume, and mute. Wave 2 added cards, profiles, and ports — making bluetooth audio profile switching a one-keystroke operation. Wave 3 added per-application streams so you can move Spotify from your laptop speakers to a bluetooth headset without touching system-wide defaults. Wave 4 (this one) replaces the polling tick with live PulseAudio property change subscription and adds the inline VU meters next to every device row.
