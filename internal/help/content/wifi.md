# Wi-Fi

Everything you need to manage wireless on this machine: see which adapters are installed, check what you're connected to, scan for networks, connect, disconnect, manage saved profiles, and run the adapter as a hotspot.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

Dark detects your wireless hardware through `sysfs` and fills in the live state by talking to `iwd` over D-Bus. The page lays out in sections from top to bottom:

1. A one-line **Wi-Fi toggle** showing whether the radio is on or off
2. A **group box** called **Adapters** listing every wireless interface on the machine
3. When you drill into an adapter, three more group boxes appear below it: **Details**, **Networks**, **Known Networks**, and (on hardware that supports it) **Access Point**

If your machine has exactly one wireless adapter, dark automatically drills into it on the first page load so you see everything at once.

## The Wi-Fi toggle

The top line shows `󰖩  Wi-Fi  On` in green when the radio is active, or `󰖪  Wi-Fi  Off` in red when it's disabled. Press `w` anywhere on the Wi-Fi section to toggle. When the radio is off, the Adapters table still shows your hardware but every other box goes empty — there's nothing to talk to until you turn it back on.

Turning the radio off actually writes `Powered = false` on iwd's phy-level Adapter object, which tears down every interface on that radio at the kernel level. It's equivalent to `rfkill block wifi` but scoped to this specific phy. Turning it back on triggers auto-reconnect to whatever known network has `AutoConnect = true`.

## The Adapters table

Each row represents one wireless interface. The columns:

- **Name** — the kernel interface name, usually `wlan0`.
- **Mode** — the operating mode: `station` (normal client), `ap` (hotspot), or `ad-hoc` (legacy peer-to-peer). Most of the time this reads `station`.
- **Powered** — `On` or `Off`. Mirrors the radio toggle.
- **State** — the station state from iwd: `Connected`, `Disconnected`, `Connecting`, `Disconnecting`, or `Roaming`. `Connected` is rendered in the accent color.
- **Scanning** — `Yes` while a scan is in progress, `No` otherwise. iwd scans on its own schedule plus whenever you press `s`.
- **Frequency** — the frequency of the current association in GHz (e.g. `5.24 GHz`). Empty when not connected.
- **Security** — the link-layer security of the current association, e.g. `WPA3-Personal`, `WPA2-Personal`, `Open`. Empty when not connected.

A leading `▸` marker shows which adapter row is selected when the content region has focus. On a single-adapter machine that's always `wlan0`.

## Navigating — focus and the key flow

Dark uses a modal focus model. When you first arrive on the Wi-Fi section, the **sidebar** has focus and `j`/`k` moves between sidebar sections. To interact with the content:

1. Press `enter` to move focus into the content region. The Adapters box border lights up in accent blue to show it has focus.
2. Press `enter` again to drill into the selected adapter. The Details, Networks, Known Networks, and Access Point boxes all appear.
3. Inside the drill-down, press `tab` to cycle focus between the **Networks** sub-table and the **Known Networks** sub-table. Whichever one is focused gets the accent border.
4. `j` / `k` moves the selection within the focused sub-table.
5. `esc` backs out one level at a time: first it closes the drill-down, then it returns focus to the sidebar, then from there it quits the app.

`q` and `ctrl+c` quit immediately from anywhere.

## The Details box

Everything the OS knows about the currently associated network and its kernel-level configuration. Fields:

- **SSID** — the network name you're associated with.
- **BSSID** — the MAC address of the specific access point radio you're talking to. When an SSID has multiple APs (mesh, multi-band router), this changes as you roam.
- **MAC** — your own hardware address.
- **Driver / Vendor / Model** — chip info from sysfs and iwd's Adapter object. Examples: `mt7921e`, `MEDIATEK Corp.`, `MT7921K (RZ608) Wi-Fi 6E 80MHz`.
- **IPv4 / IPv6** — addresses assigned to the interface from netlink, with CIDR mask.
- **Gateway** — the default route for this interface, parsed from `/proc/net/route`.
- **DNS** — nameserver list from `/etc/resolv.conf`. If you're using systemd-resolved, this will typically show `127.0.0.53` (the stub resolver).
- **Channel** — the current Wi-Fi channel number.
- **Signal** — current RSSI in dBm, optionally the average RSSI, and a rolling sparkline of the last ~10 minutes of samples. The sparkline scale is pinned to -30 dBm (excellent, `█`) and -90 dBm (unusable, `▁`) so you can compare strength visually across sessions.
- **Link** — the PHY mode and current TX / RX bitrates, e.g. `802.11ac  (TX 5.9 Mbps · RX 3.5 Mbps)`.
- **Traffic** — cumulative RX and TX byte totals since the interface came up, read from sysfs `statistics/` files.
- **Rate** — current throughput in bits per second, computed from the byte-counter delta across successive snapshots.
- **Connected** — how long the current association has been up.

Everything except `IPv4`, `IPv6`, `Gateway`, `DNS`, `Traffic`, and `Rate` comes from iwd's `Station` and `StationDiagnostic` D-Bus interfaces. The rest is read directly from the kernel via sysfs and `/proc`.

## The Networks box

Every SSID iwd currently knows about on this adapter, ordered by signal strength. The data comes from iwd's `GetOrderedNetworks()` method which returns whatever iwd has cached from its most recent scan. Columns:

- **SSID** — the network name.
- **Security** — `WPA2/3` (iwd doesn't distinguish), `Open`, `Enterprise`, or `WEP`.
- **Signal** — RSSI in dBm as a signed number.
- **Bars** — the same RSSI rendered as a Unicode bar graph for quick visual comparison.
- **BSS** — the number of access points broadcasting this SSID (mesh systems, multi-band routers).

Row markers on the left:

- `󰖩` (the Wi-Fi glyph, accent color) marks the network you're currently associated with.
- `▸` (accent triangle) marks the selection cursor when this sub-table has focus.

If both apply to the same row you'll see both glyphs side by side.

## The Known Networks box

Every saved profile iwd has on disk. These are networks you've previously connected to — the SSID and credentials live at `/var/lib/iwd/<name>.psk` (or `.open` / `.8021x`), and iwd auto-connects to them when they're in range and `AutoConnect` is enabled. Columns:

- **SSID** — the network name.
- **Security** — `WPA2/3`, `Open`, `Enterprise`, or `WEP`.
- **Auto** — `Yes` if iwd should automatically join this network when it sees it, `No` if not.
- **Hidden** — `Yes` if the network is marked as hidden (doesn't broadcast its SSID). The default for anything you joined via the Networks box is `No`.
- **Last seen** — when iwd most recently associated with this profile. Format is relative: `3m ago`, `2h ago`, `5d ago`, or falls back to a date for older entries.

A `▸` marker shows the selection when this sub-table has focus.

## The Access Point box

Only shown if your adapter hardware lists `ap` in its SupportedModes. On adapters that only support station mode, this box is hidden entirely because there's nothing it could do.

When the AP is **stopped**, the box reads `not running — press p to start a hotspot`.

When the AP is **running**, the box shows `Status: Running`, the hotspot SSID, its operating frequency, and the current Mode, plus a reminder to press `p` to stop.

See the Hotspot section below for the full flow.

## Actions — the complete keybinding reference

### From the sidebar (before you've pressed enter)

- `j` / `k` or `↑` / `↓` — move between sidebar sections
- `enter` — move focus into the Adapters box
- `w` — toggle the Wi-Fi radio
- `h` — open the hidden network dialog (works from any focus level)
- `p` — start or stop the hotspot on the selected adapter (works from any focus level)
- `?` — open this help drawer
- `ctrl+r` — rebuild dark and hot-reload
- `esc` or `q` — quit

### From Adapters focus (after one `enter`)

- `j` / `k` — move the adapter row selection
- `enter` — drill into the selected adapter (opens Details, Networks, Known Networks, and possibly Access Point)
- `w` — toggle the Wi-Fi radio
- `h` — hidden network dialog
- `p` — start / stop hotspot
- `esc` — return focus to the sidebar

### From the drill-down (after two `enter` presses)

These keys operate on whichever sub-table currently has focus, which is shown by the accent border color on the Networks or Known Networks box.

- `tab` — cycle focus between Networks and Known Networks
- `j` / `k` — move the selection within the focused sub-table
- `c` — connect to the highlighted network. For known networks and open networks this connects immediately. For unknown WPA networks a password dialog pops up first.
- `d` — disconnect the adapter from whatever it's currently associated with
- `f` — forget the highlighted network (Known Networks sub-table only)
- `a` — toggle AutoConnect on the highlighted saved network (Known Networks sub-table only)
- `s` — trigger a live scan and refresh the Networks list
- `h` — open the hidden network dialog
- `p` — start / stop hotspot
- `w` — toggle radio
- `esc` — close the drill-down and return to Adapters focus

### Universal

- `ctrl+c` — quit from anywhere, including inside a dialog
- `?` — open this help drawer
- `ctrl+r` — rebuild in place

## Dialogs

Three places in the Wi-Fi flow open a modal dialog: connecting to an unknown network, connecting to a hidden network, and starting a hotspot. They all use the same component.

- `tab` or `↓` — move to the next field
- `shift+tab` or `↑` — move to the previous field
- `backspace` — delete the previous character in the active field
- Any printable key types into the active field
- `enter` — submit the dialog and dispatch the command
- `esc` — cancel without doing anything
- Password fields show typed characters as `•` bullets

When a dialog is open, every key event goes to the dialog. The sidebar, the sub-tables, and the global shortcuts all pause until you either submit or cancel.

## Common tasks — how to actually do things

### Connect to a network I've used before

Dark connects automatically to saved networks via iwd's AutoConnect. If you're disconnected and a known network is in range, it should just reconnect on its own within a few seconds. If it doesn't:

1. Drill in with two `enter` presses.
2. The Networks sub-table is focused by default. Arrow to the network you want (it'll have the `󰖩` glyph if you're already on it).
3. Press `c`. Because the network has a saved profile, dark issues the connect without prompting for credentials.

### Connect to a brand-new network

1. Drill in so you can see the Networks sub-table.
2. Arrow to the new network.
3. Press `c`. Because there's no saved profile, dark opens a **Connect to `<SSID>`** dialog.
4. Type the passphrase. Press `enter`.
5. The Networks box shows `working…` in accent while the daemon negotiates with iwd.
6. On success the network appears in Known Networks with a fresh profile, and you're connected. On failure the dialog's action error line shows what iwd reported.

### Connect to a hidden network (one that doesn't broadcast its SSID)

1. From anywhere on the Wi-Fi page, press `h`.
2. The **Connect to hidden network** dialog opens with two fields: SSID and Passphrase.
3. Type the exact SSID. Press `tab` to move to the Passphrase field. Leave blank for open hidden networks.
4. Press `enter`. Dark calls iwd's `ConnectHiddenNetwork` which probes for the hidden SSID, connects if found, and saves a profile.

### Disconnect

Press `d` from anywhere in the drilled-in view. Always acts on the currently associated network — no selection needed. Afterwards iwd may auto-reconnect to the same network because it's still marked `AutoConnect = Yes`. To prevent that, see the next task.

### Stop auto-reconnecting to a network without forgetting it

1. Drill in. Press `tab` to focus the Known Networks sub-table.
2. Arrow to the network.
3. Press `a`. The Auto column flips from `Yes` to `No`. iwd will no longer join this network on its own but the credentials are still saved, so you can still connect manually with `c`.

### Forget a saved network

1. Drill in. `tab` to focus Known Networks.
2. Arrow to the network.
3. Press `f`. iwd deletes the profile file from `/var/lib/iwd/`. If you're currently connected to that network, iwd disconnects you. The row disappears from Known Networks and the corresponding row in the Networks list flips from Known back to Unknown.

### Turn the Wi-Fi radio off

Press `w`. The radio powers down, the toggle switches to `󰖪 Wi-Fi Off` in red, and the Networks / Known Networks sub-tables empty out. Press `w` again to turn it back on. iwd will auto-reconnect to whichever known network is in range.

### Start a hotspot (AP mode)

This turns your adapter into an access point other devices can join. While the hotspot is running, this machine is **not** connected to the outside internet via Wi-Fi on this adapter — you're providing wireless service, not consuming it. If you need upstream internet, use ethernet or a second wireless adapter.

1. Make sure the selected adapter supports AP mode — the Access Point box only shows when it does.
2. Press `p`. The **Start access point** dialog opens.
3. Type the SSID you want to broadcast. Press `tab`. Type a passphrase (iwd requires 8+ characters for WPA).
4. Press `enter`. Dark switches the device from station mode to AP mode (which disconnects from your current Wi-Fi) and starts the access point.
5. The Access Point box flips to **Running** with the SSID and frequency. The Adapters table shows `Mode: ap`. The Networks and Known Networks sub-tables go empty because iwd's Station interface isn't present on an AP device.

### Stop the hotspot

Press `p` again. Dark stops the AccessPoint, switches the device back to station mode, and iwd auto-reconnects to your normal Wi-Fi within a few seconds.

## Status line meanings

When the daemon is actively working on something, the Networks box shows a status line above the table:

- `scanning…` (accent) — a live scan is in flight
- `working…` (accent) — a connect / disconnect / forget / AP command is in flight
- `scan failed: <reason>` (red) — the last scan errored, usually "timeout" or "not scanning"
- `action failed: <iwd error>` (red) — the last action errored, with whatever iwd reported

The status line clears as soon as the command finishes successfully.

## Data sources, for the curious

Everything on this page comes from the daemon (`darkd`), which reads:

- **Sysfs** (`/sys/class/net/`) for adapter enumeration, driver, MAC, phy name, and traffic counters.
- **System D-Bus** via iwd's `net.connman.iwd` service for mode, powered, state, scanning, SSID, scan results, known networks, AutoConnect, and the full `StationDiagnostic.GetDiagnostics()` output.
- **The Go `net` package** (netlink under the hood) for IPv4 and IPv6 addresses.
- **`/proc/net/route`** for the default gateway.
- **`/etc/resolv.conf`** for DNS servers.
- **iwd's `AccessPoint` interface** for AP mode state and actions.

The daemon registers an `iwd.Agent` implementation at startup so unknown-network connects can supply a passphrase without storing it on disk from dark's side. iwd itself writes the profile to `/var/lib/iwd/` when the connect succeeds.

Dark publishes a fresh wifi snapshot on `dark.wifi.adapters` every 30 seconds, and pushes an updated snapshot immediately after every successful action. `ctrl+r` rebuilds and reloads dark if you're iterating on it.

## Backend notes

Right now dark talks to **iwd**. The code is structured behind a `Backend` interface so NetworkManager support can be added as an additional file without touching the TUI. If you switch to NetworkManager before that implementation lands, the Adapters table will still show your hardware but the action keys will report `operation not supported by this wifi backend`.

Regulatory domain, supported bands (2.4 / 5 / 6 GHz), and the maximum 802.11 standard the hardware can do aren't exposed yet — those need raw `nl80211` netlink queries which aren't wired up. The currently associated PHY mode is shown on the `Link` row of Details.
