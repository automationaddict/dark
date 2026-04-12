# Bluetooth

Manage the host's Bluetooth controller and the devices it knows about: see the adapter's state, scan for nearby devices, pair, connect, disconnect, trust, and unpair.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

Dark talks to **BlueZ** over the system D-Bus (`org.bluez`). The page lays out in sections from top to bottom:

1. A one-line **Bluetooth toggle** showing whether the controller's radio is on or off
2. A **group box** called **Adapters** listing every bluetooth controller on the machine
3. When you drill into an adapter, a **Details** box and a **Devices** box appear below it

If your machine has exactly one powered controller, dark automatically drills into it on the first page load.

## The Bluetooth toggle

The top line shows `󰂯  Bluetooth  On` in green when the radio is active, or `󰂲  Bluetooth  Off` in red when it's disabled. Press `w` anywhere on the Bluetooth section to toggle. Turning the radio off writes `Powered = false` on the BlueZ Adapter1 object, which disconnects every device and stops any in-flight discovery.

## The Adapters table

Each row represents one controller. Columns:

- **Name** — the BlueZ adapter name, usually `hci0`.
- **Alias** — the friendly name broadcast to other devices. Settable via BlueZ; defaults to the hostname.
- **Address** — the controller's own MAC.
- **Powered** — `On` or `Off`. Mirrors the radio toggle.
- **Discoverable** — `Yes` when the adapter is advertising itself so other devices can see it.
- **Scanning** — `Yes` while a discovery scan is in progress. Toggled by `s`.

A leading `▸` marker shows which adapter row is selected when the content region has focus.

## Navigating — focus and the key flow

Dark uses a modal focus model:

1. Press `enter` to move focus into the content region. The Adapters box border lights up in accent blue.
2. Press `enter` again to drill into the selected adapter. Details and Devices boxes appear.
3. `j` / `k` moves the selection within the Devices list.
4. Press `enter` a third time on a highlighted device to expand the **Device Info** panel, which replaces the Devices list with the full property readout for that one device.
5. `esc` backs out one level at a time: Device Info → Devices list → Adapters → sidebar → quit.

`q` and `ctrl+c` quit from anywhere.

## The Details box

Adapter metadata BlueZ exposes:

- **Name / Alias / Address** — identifiers.
- **Powered / Discoverable / Pairable / Discovering** — live state flags.
- **Devices** — how many Device1 objects BlueZ currently has cataloged under this adapter.

## The Device Info panel

Press `enter` on a highlighted device in the Devices list to expand it. The Devices table is replaced by a single-device property readout that covers everything BlueZ knows:

- **Name / Alias** — the raw advertised name and the friendly alias (may differ if the user or another tool has overridden it).
- **Address** — MAC plus BlueZ address type (`public` for Classic / BR-EDR devices, `random` for LE devices using a random resolvable address).
- **Type** — decoded BlueZ icon hint.
- **Class** — the 24-bit Class of Device value in hex plus the decoded major class (Computer, Phone, Audio/Video, Peripheral, etc.). Zero on LE-only devices.
- **Modalias** — the vendor/product identifier in USB-style format (`usb:v05ACp8001d0004`), useful for looking up exact hardware.
- **Paired / Bonded / Trusted / Blocked / Connected / LegacyPairing / Services Resolved** — live state flags.
- **RSSI / Tx Power / Battery** — radio and power readings when BlueZ has them.
- **Services** — the device's advertised UUIDs. Dark knows the common profile names (A2DP, AVRCP, HFP, HID over GATT, Battery Service, etc.) and shows them next to the raw UUID. Unknown UUIDs fall through as raw hex so nothing is silently dropped.

Action keys work on the same selected device while the info panel is open — `c` connects, `d` disconnects, `t` trusts, `b` blocks, `u` unpairs. Press `esc` to return to the Devices list.

## The Devices box

Every device BlueZ knows about on this adapter, currently visible or previously paired. Ordered by: connected first, then paired, then by strongest RSSI, then by name. Columns:

- **Name** — the device's friendly name (Alias in BlueZ).
- **Address** — the device MAC.
- **Type** — a short label derived from BlueZ's Icon hint (`headset`, `keyboard`, `phone`, etc.).
- **Paired** — `Yes` if there's a bond on disk.
- **Trusted** — `Yes` if the adapter will auto-authorize services without prompting.
- **RSSI** — signal strength in dBm when BlueZ has a recent measurement; `—` otherwise.
- **Battery** — battery percentage when the device exposes the BlueZ Battery1 interface; `—` otherwise.

Row markers:

- `󰂱` (accent) marks devices currently **connected**.
- `▸` marks the selection cursor when the content region is focused.

## Actions — keybinding reference

### From the sidebar

- `j` / `k` — move between sidebar sections
- `enter` — move focus into the Adapters box
- `w` — toggle the Bluetooth radio
- `?` — open this help drawer
- `esc` or `q` — quit

### From Adapters focus

- `j` / `k` — move the adapter row selection
- `enter` — drill into the selected adapter
- `w` — toggle the radio
- `s` — start or stop discovery
- `y` — toggle Discoverable (advertise this controller so other devices can see it)
- `a` — toggle Pairable (accept or refuse incoming pair requests)
- `r` — rename the adapter (edit its Alias)
- `R` (shift+r) — reset the alias back to the system default (BlueZ usually uses the hostname)
- `T` (shift+t) — set the discoverable timeout in seconds
- `F` (shift+f) — set the scan filter (transport, RSSI floor, name pattern)
- `esc` — return focus to the sidebar

### From the drill-down

Keys operate on the highlighted device in the Devices list:

- `j` / `k` — move the device selection
- `s` — start or stop discovery (scanning)
- `c` — connect to the highlighted device
- `d` — disconnect
- `p` — pair with the highlighted device (pops a PIN dialog if the device is legacy)
- `x` — cancel an in-flight pair
- `u` — unpair (removes the device from BlueZ)
- `t` — toggle Trusted
- `b` — toggle Blocked (refuse all interaction with the highlighted device)
- `y` — toggle Discoverable on the adapter
- `a` — toggle Pairable on the adapter
- `r` — rename the adapter
- `R` — reset the alias to system default
- `T` — set the discoverable timeout
- `F` — set the scan filter
- `w` — toggle the radio
- `esc` — close the drill-down

### Universal

- `ctrl+c` — quit from anywhere
- `?` — open this help drawer
- `ctrl+r` — rebuild in place

## Common tasks

### Pair a new device

1. Put the device into pairing mode.
2. In dark, drill into the adapter and press `s` to start discovery. The Devices list updates as BlueZ sees new devices.
3. Arrow to the device, press `p`.
4. For modern devices, dark auto-confirms the numeric-comparison prompt (the standard SSP flow for phones, headphones, and most keyboards). For **legacy** devices (pre-SSP hardware like old speakers, car kits, and cheap headsets) BlueZ flags the device with LegacyPairing and dark pops a PIN dialog. Type the PIN the device expects — most hardware ships with `0000` or `1234` — and press `enter`. The PIN is handed to BlueZ via the agent and the pair proceeds.
5. On success, Paired flips to `Yes`. Dark does not automatically connect after pairing; press `c` if you want to connect immediately.
6. If the pair stalls or you picked the wrong device, press `x` to cancel. BlueZ aborts the in-flight `Device1.Pair` and the device stays unpaired.

### Connect to a paired device

Arrow to the device and press `c`. If the device is already paired and trusted, the connect happens without prompts.

### Trust a device so future connects are silent

1. Select the device in the Devices list.
2. Press `t`. The Trusted column flips to `Yes`. BlueZ will now auto-authorize service connections from this device without calling our agent.

### Disconnect a device

Arrow to the device and press `d`. The bond stays in place so you can reconnect later with `c`.

### Forget a device

Arrow to it and press `u` (unpair). This calls BlueZ's `RemoveDevice`, which deletes the bond and removes the Device1 object entirely. Reconnecting requires a fresh pair.

### Turn the radio off

Press `w`. Everything disconnects, discovery stops, and the Devices list empties. Press `w` again to bring it back.

### Make this machine visible to other devices

Press `y`. The adapter's Discoverable flag flips to `Yes` and the controller starts broadcasting its presence so other devices can find it for pairing. Press `y` again to hide. BlueZ has a default discoverable timeout (usually 3 minutes) after which it hides itself again automatically — that timeout is a BlueZ setting, not a dark setting.

### Rename the adapter

Press `r`. A dialog opens prefilled with the current Alias. Type the new name and press `enter`. The change persists — BlueZ writes it to disk and continues to advertise the new name on subsequent power cycles.

### Refuse incoming pair requests

Press `a` to flip the adapter's Pairable flag. When Pairable is `No`, BlueZ rejects any incoming pair attempt from another device even if the adapter is Discoverable. Pairable controls "can others pair with me", Discoverable controls "can others see me" — they're independent.

### Set how long Discoverable stays on

Press `T` (shift+t). A dialog opens prefilled with the current `DiscoverableTimeout` in seconds. Type a new value and press `enter`. `0` means "never time out" — the adapter stays discoverable until you explicitly toggle it off. BlueZ defaults this to 180 seconds (three minutes). The decoded timeout is shown in the Details box next to the Discoverable row.

### Reset the adapter alias

Press `R` (shift+r). Dark writes an empty string to `Adapter1.Alias`, which BlueZ treats as a request to clear the custom alias and fall back to the system default (usually the hostname). No dialog, no confirmation — press `r` afterwards if you want to pick a new custom name instead.

### Filter the scan results

Press `F` (shift+f). A dialog opens for three `Adapter1.SetDiscoveryFilter` parameters:

- **Transport** — `auto` (the default, scan both Classic and LE), `bredr` (Classic only), or `le` (LE only). Useful for cutting LE noise when pairing a Classic headset, or the reverse.
- **RSSI floor** — a dBm value like `-70`. Devices weaker than this are dropped from the scan results so you only see nearby devices. Leave blank to disable the floor.
- **Name pattern** — a substring/prefix match on the advertised name. Leave blank to disable.

Dark remembers the filter values across subsequent scans within a session. Submitting all-empty clears the filter (equivalent to `SetDiscoveryFilter({})`).

Per BlueZ, filters only take effect on the *next* StartDiscovery. If a scan is already running when you change the filter, stop and restart it with `s` to apply.

### Block a device

Press `b` on the highlighted device. BlueZ sets the Blocked flag and refuses all future interactions with that device — connects, pairs, and even advertising traffic are ignored until you unblock. Useful when a nearby speaker keeps trying to pair with your headphones every time you're in range. Press `b` again on the same device to unblock.

## Status line meanings

When the daemon is working on something, the Devices box shows a status line above the list:

- `scanning…` (accent) — discovery is running
- `working…` (accent) — a connect / disconnect / pair / unpair / trust command is in flight
- `action failed: <bluez error>` (red) — the last action errored, with whatever BlueZ reported

The status line clears when the next snapshot arrives.

## Data sources

Everything on this page comes from `darkd`, which reads:

- **System D-Bus** at `org.bluez` for adapter enumeration, device enumeration, and every action. BlueZ's `ObjectManager` gives us the full object tree in one call.
- **BlueZ's `Battery1`** interface when devices expose it.
- **BlueZ's `AgentManager1`** — dark registers its own Agent with `KeyboardDisplay` capability at startup so BlueZ routes pairing prompts to us. The current agent auto-confirms numeric-comparison pairings and refuses legacy PIN entry.

The daemon publishes a fresh bluetooth snapshot on `dark.bluetooth.adapters` every 15 seconds, and pushes an updated snapshot immediately after every successful action.

## Backend notes

Dark currently talks to BlueZ. The code is structured behind a `Backend` interface, but BlueZ is effectively the only bluetooth stack on Linux — there's no realistic second backend to swap in. Legacy PIN-based pairing, audio profile switching (A2DP / HSP / HFP), GATT browsing, and OBEX file transfer are deferred to later passes.
