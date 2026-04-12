# Network

Read-only view of the host's network state — every interface, its addresses, link state, traffic counters, the system DNS configuration, and the kernel routing table.

Press `?` at any time to open this help. Press `esc` to close it.

## What "Network" covers (and what it doesn't)

The Network section is for **layer 3 and the wired side of layer 2** — IP addressing, routing, DNS, and physical/virtual interface state for everything that isn't a wireless association. The Wi-Fi section already handles wireless association, scans, and known networks.

This is the **Tier 1** read-only surface. Tier 3 will add a pluggable backend for *changing* configuration (static IPs, DNS overrides, bringing interfaces up and down), at which point dark will need to know which network manager you're running — systemd-networkd, NetworkManager, dhcpcd, or none. Until then everything on this page is observation only.

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
- `esc` — return focus to the sidebar
- `?` — open this help drawer

There are no action keys yet because Tier 1 is read-only. Tier 3 will add operations like "bring this interface up", "release/renew DHCP", "edit static IP", "change DNS servers", and at that point dark will need to know which network manager is configuring your system so it can write the right config files.

## Tier roadmap

- **Tier 1 (this)** — read-only kernel scrape. Works on every Omarchy install regardless of which network manager (if any) is configuring things.
- **Tier 2** — basic mutating operations that don't need a config daemon: bring interfaces up/down via netlink, trigger DHCP renewal where we can detect the client, add temporary static addresses.
- **Tier 3** — pluggable Backend interface for persistent configuration. systemd-networkd, NetworkManager, and dhcpcd implementations. Detection at startup picks whichever is running.
