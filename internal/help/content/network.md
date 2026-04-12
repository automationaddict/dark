# Network

Read-only view of the host's network state — every interface, its addresses, link state, traffic counters, the system DNS configuration, and the kernel routing table.

Press `?` at any time to open this help. Press `esc` to close it.

## What "Network" covers (and what it doesn't)

The Network section is for **layer 3 and the wired side of layer 2** — IP addressing, routing, DNS, and physical/virtual interface state for everything that isn't a wireless association. The Wi-Fi section already handles wireless association, scans, and known networks.

Reading state always works regardless of what's managing your network — the kernel scrape doesn't depend on any daemon. Mutating operations go through a pluggable Backend interface with implementations for **systemd-networkd** and **NetworkManager**. Whichever is running on your system gets picked at startup; if neither is running, the section stays read-only and the action keys report `operation not supported by this network backend`. The active backend is shown in the focus hint at the bottom of the page.

## Where the data comes from

Tier 1 reads the kernel directly with no daemon dependency, so it works on any Omarchy install regardless of what's managing your network:

- **`/sys/class/net/<iface>/`** — driver, MAC, MTU, link state, carrier, traffic counters, speed, duplex.
- **The Go `net` package** (netlink under the hood) — interface enumeration and IPv4/IPv6 address binding.
- **`/proc/net/route`** — IPv4 routing table.
- **`/proc/net/ipv6_route`** — IPv6 routing table.
- **`/etc/resolv.conf`** — DNS servers and search domains. On systems running systemd-resolved this is usually a symlink to `/run/systemd/resolve/stub-resolv.conf` listing `127.0.0.53`; that stub address is the truth from the perspective of every application on the system.

The daemon publishes a fresh snapshot every 10 seconds plus an on-demand reply for the initial load.

## The Interfaces table

Every network device the kernel knows about, sorted with real hardware first (ethernet, then wireless), then virtual interfaces (bridges, bonds, tun/tap, veth), then loopback last. Columns:

- **Name** — kernel interface name (`enp3s0`, `wlan0`, `lo`, `docker0`, etc.).
- **Type** — `ethernet`, `wireless`, `bridge`, `bond`, `virtual`, `loopback`. Detected from sysfs by checking for `wireless/`, `bridge/`, `bonding/` subdirectories or the absence of a `device/` symlink.
- **State** — operstate from sysfs: `up`, `down`, `dormant`, `lowerlayerdown`, `unknown`. When the interface is administratively up but no carrier is detected, state reads `up · no link` (e.g. an ethernet port with no cable plugged in).
- **IPv4 / IPv6** — primary global-scope address. Link-local IPv6 (`fe80::/10`) is excluded from this column to keep it readable; the Details box shows everything.
- **Speed** — link speed in Mbps or Gbps when sysfs reports it. Wireless and most virtual interfaces don't, so they show `—`.
- **Rate** — current RX↓ / TX↑ throughput in bits per second, computed from the byte-counter delta between snapshots.

A leading `▸` marks the highlighted row when the content region has focus.

## The Details box

Reflects whichever interface row is currently highlighted. Shows everything in the table plus:

- **Driver** — kernel module backing the device (`iwlwifi`, `e1000e`, `r8169`, etc.). Empty for virtual interfaces.
- **MAC** — hardware address.
- **MTU** — link MTU in bytes.
- **Link** — carrier state, speed, and duplex when known. Reads `no link` (dimmed) when carrier is down.
- **IPv4 / IPv6** — full address list with CIDR prefixes. Link-local and host-scope addresses are tagged with their scope in parentheses.
- **RX/TX bytes** — cumulative traffic since the interface came up.
- **RX/TX packets** — cumulative packet counts.
- **Rate** — same throughput readout as the table column.

When an active backend recognizes the interface, a second section appears below those rows with manager-specific state:

- **Managed by** — `systemd-networkd` or `NetworkManager`. Tells you which backend is in charge of this device's configuration.
- **Admin state** — networkd's `AdministrativeState` (`configured` / `configuring` / `failed` / `unmanaged`) or NM's `Device.State` (`activated` / `disconnected` / `failed` / etc.). This is the manager's view of "is this device set up", which is distinct from the kernel's `state` column above. A device can be `up · routable` at the kernel level while showing `failed` here if the manager hit a configuration error after the link came up.
- **Online state** — networkd's `OnlineState` (`online` / `partial` / `offline`), reported separately from `AdministrativeState`. NM doesn't expose a directly equivalent property.
- **Config** — the `.network` file path (systemd-networkd) or active connection name (NetworkManager) currently driving this interface. The fastest way to find "where is this device's settings actually coming from".
- **DHCPv4 / DHCPv6** — DHCP client state per family when DHCP is active (`bound` / `requesting` / `stopped` / etc.). Hidden when there's nothing useful to report.
- **DNS (link)** — DNS servers configured by the manager *for this specific interface*, which may differ from the global DNS box at the bottom of the page (especially when systemd-resolved is doing per-link routing).
- **Domains** — search/route domains the manager attached to this link.
- **Required** — networkd's `RequiredForOnline` flag or NM's `Autoconnect` — does the manager consider this interface required for the system to be considered "online".

When no backend is detected, this section is omitted entirely. When the backend simply doesn't recognize the interface (loopback, virtual interfaces it doesn't manage), it's also omitted for that row. The kernel-level details above always render regardless.

## The DNS box

The system resolver configuration:

- **Servers** — every `nameserver` line from `/etc/resolv.conf`. On systemd-resolved systems this is `127.0.0.53` (the stub resolver). The actual upstream servers in that case are visible to `resolvectl`, not in the file — but `127.0.0.53` is what every application on your machine actually queries, so showing it is correct.
- **Search** — search domains and the legacy `domain` directive.
- **Source** — the path the data was read from. Resolved through any symlink so you can tell at a glance whether you're looking at a hand-edited file or a daemon-managed one.

The DNS box only renders when there's something to show.

## The Routes box

Every route in the kernel routing table, sorted with default routes (the gateway-of-last-resort) at the top. Columns:

- **Destination** — `default` for the gateway-of-last-resort, otherwise a CIDR like `192.168.1.0/24`.
- **Gateway** — next-hop address. Empty for on-link routes.
- **Interface** — which interface the route is bound to.
- **Metric** — kernel routing metric. `—` when zero.
- **Family** — `ipv4` or `ipv6`.

Default routes are highlighted in the accent color since they're the most operationally important entries — knowing your default gateway is correct is usually what you actually came to this page to verify.

## Navigation

- `enter` — move focus into the Interfaces box
- `j` / `k` — walk the interface row selection
- `r` — reconfigure the highlighted interface (re-read configuration and re-apply)
- `h` — switch the highlighted interface to DHCP
- `e` — open the static IPv4/IPv6 editor for the highlighted interface
- `t` — drill into the routes management view for the highlighted interface
- `R` (shift+r) — reset the highlighted interface back to system defaults (deletes the dark-managed config)
- Inside the routes drill-in: `a` add a route, `d` delete the highlighted route, `j`/`k` navigate, `esc` back out
- `esc` — return focus to the sidebar
- `?` — open this help drawer

## Editing IPv4 configuration

Four keys cover the common cases:

- `h` — switch to DHCP. One keystroke, no dialog. The active backend writes a config that asks DHCP for an address.
- `e` — open a dialog to set a static address (IPv4 and optionally IPv6) plus everything else dark manages for the interface. The fields cover both families and shared concerns. When dark already has a config for this interface, the dialog prefills from dark's own previously-written values rather than from kernel state, so what you edit is exactly what you wrote.
- `t` — drill into the routes management view for the highlighted interface. See "Static routes" below.
- `R` (shift+r) — reset the interface back to system defaults. Deletes the dark-managed config file (still via the privileged helper, so you'll see a polkit prompt) and triggers a reload + reconfigure so whatever else matches the link — a distro default `.network` file or systemd-networkd's implicit fallback — takes over. The "I changed my mind" verb. No dialog, no confirmation; the action is reversible by pressing `e` or `h` again.

### The static editor (`e`) fields

| Field | Required when | Notes |
|---|---|---|
| IPv4 address | always (static mode) | CIDR format like `192.168.1.10/24` |
| IPv4 gateway | optional | leave blank for on-link only |
| IPv6 mode | always | one of `dhcp`, `static`, `ra`, or blank for "leave IPv6 unspecified" |
| IPv6 address | required if v6 mode = static | CIDR like `2001:db8::1/64` |
| IPv6 gateway | optional | usually a link-local address like `fe80::1` |
| DNS servers | optional | comma-separated, IPv4 *and* IPv6 nameservers can both be in this list |
| Search domains | optional | comma-separated |
| MTU bytes | optional | blank = kernel/driver default |

The IPv6 mode field is the entry point for IPv6 configuration. Set it to `dhcp` for DHCPv6, `static` for a hand-set address, `ra` to accept router advertisements (the typical SLAAC case for most home networks), or leave it blank to write nothing IPv6-related at all (systemd-networkd then uses its own defaults for the family).

### Static routes (`t` drill-in)

Press `t` to drill into the routes management view for the highlighted interface. The view shows dark's currently-managed route list as a table with destination, gateway, and metric columns. Inside the drill-in:

- `j` / `k` — move the route selection cursor
- `a` — add a new route (opens a small 3-field dialog)
- `d` — delete the highlighted route
- `esc` — back out to the regular Network view

Each route's destination is a CIDR like `10.0.0.0/8` or `0.0.0.0/0` for a default route. Gateway is optional — leaving it blank produces an on-link route. Metric is also optional and determines the route's priority when multiple routes match (lower wins).

The drill-in only opens for interfaces that already have a dark-managed config. If you press `t` on an interface dark hasn't touched, you'll get a notification telling you to set up basic IPv4 first via `h` or `e`.

To clear all dark-managed routes at once, use `R` (reset) in the regular Network view — that removes the entire dark config including the route list.

### How privilege escalation works

`systemd-networkd`'s configuration lives in files under `/etc/systemd/network/`, which are owned by root. There is no D-Bus method for editing them — networkd's writable D-Bus surface is only for runtime operations like reload and reconfigure. So when dark needs to write a new `.network` file, it has to do it as root.

Dark handles this with a small companion binary called `dark-helper`. The helper is intentionally minimal — its entire job is to validate a file path against a strict whitelist (must be under `/etc/systemd/network/`, must end in `.network`, must be directly in that directory with no subdirectory traversal, content read from stdin, max 64 KiB) and then atomically write it. It accepts no other operations and reads no other paths.

When you press `e` and submit the dialog, this happens:

1. `darkd` builds the `.network` file content from your dialog input.
2. `darkd` invokes `pkexec dark-helper write-network-file /etc/systemd/network/50-dark-<iface>.network` with the content on stdin.
3. `pkexec` triggers your session's polkit agent (the GUI dialog you've seen before from other Linux apps). You enter your password. polkit checks the rules.
4. If polkit approves, the helper runs as root, writes the file, and exits.
5. `darkd` then calls `Manager.Reload()` and `Manager.ReconfigureLink()` on networkd via D-Bus to pick up the new file. These calls don't need pkexec — they're polkit-protected on the daemon side and your active session has permission for them by default.

The file written is named `50-dark-<iface>.network` and starts with a header comment marking it as managed by dark. The "50-" weight puts it after distro defaults (10-, 20-) and before user overrides (90-, 99-) — the conventional middle ground for a settings tool.

If you want to revert to the system default, deleting the file (which you can do via the helper too — that operation is wired up but no key surfaces it yet) will leave the interface back under the original config. Hand-editing the file is also fine; dark will overwrite it the next time you press `h` or `e`, but won't touch it otherwise.

`NetworkManager` doesn't have this problem because its connection profile editing is exposed via polkit-protected D-Bus calls. The NetworkManager backend handles `ConfigureIPv4` via D-Bus directly, no helper needed. (Status: that path is stubbed in this wave and will be filled in next wave.)

## The reconfigure verb

Press `r` on a highlighted interface to ask the active backend to re-read its configuration and re-apply it. This is the "kick this interface and make it try again" verb that does what most users want when something is wrong with their network — it triggers a fresh DHCP request, picks up any config changes made to `.network` files or NetworkManager profiles, and clears transient stuck state.

Behind the scenes:

- **systemd-networkd** receives `Manager.ReconfigureLink(ifindex)` over D-Bus, which re-reads the matching `.network` file and re-acquires DHCP if DHCP is configured.
- **NetworkManager** receives `Device.Reapply()` on the device, which re-applies the device's current connection profile without tearing it down.

Both calls are polkit-protected on the daemon side. Active user sessions have permission for these actions in the default polkit policy that ships with both daemons, so there's no password prompt or escalation step from dark's side — godbus issues the call, the system bus invokes polkit, polkit checks the session, and either the call succeeds or you get an `AccessDenied` error displayed inline beneath the Interfaces table.

If no backend is detected (you're running iwd-only with no wired manager, or some other unusual setup), pressing `r` will report `operation not supported by this network backend`. Reading state still works in that case — only mutations are gated.

## Wave roadmap

- **Wave 1** — read-only kernel scrape (interfaces, addresses, routes, DNS, traffic counters). Works on any system regardless of which network manager is running.
- **Wave 2** — pluggable Backend interface plus systemd-networkd and NetworkManager implementations, both via polkit-protected D-Bus. First mutating operation is `r` to reconfigure an interface.
- **Wave 3 (this)** — Backend `Augment()` populates per-interface management state from the active manager. Surfaces admin state, online state, the file/connection managing each link, per-link DNS, and DHCP client state in the Details box.
- **Wave 4** — connection-level IPv4 editing: switch between DHCP and static addressing, set static IP / gateway / DNS / search domains. systemd-networkd uses the new `dark-helper` privileged binary invoked via `pkexec` to write `.network` files. NetworkManager backend stubbed.
- **Wave 5** — MTU editing and a `R` reset key.
- **Wave 6** — route management with `[Route]` sections, file parser, `t` route text field.
- **Wave 7** — routes drill-in replacing the text encoding. Per-route add (`a`), delete (`d`), and j/k navigation in a dedicated full-screen view.
- **Wave 8 (this)** — IPv6 support. The static editor now covers both IPv4 and IPv6 families with IPv6 mode (dhcp/static/ra), address, and gateway fields. The `.network` file generator emits the right systemd-networkd keys for each family and the parser reads them back for prefill.

### Known gaps (deferred)

- **NetworkManager `ConfigureIPv4`**: the stub exists but the real D-Bus connection-profile editing is unimplemented. Will be added when there's a NM machine to test against.
- **Firewall / nftables**: separate domain, not a Network section concern.
- **VPN / WireGuard**: separate domain, likely its own future section.
- **Hostname editing**: small scope, could be added via `org.freedesktop.hostname1`.
