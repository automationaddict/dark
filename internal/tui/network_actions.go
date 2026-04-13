package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/network"
)

// NetworkActions is the set of asynchronous commands the TUI can
// dispatch at darkd to drive the network service. Mirrors the other
// section action types — the daemon side does the actual work, the
// TUI just builds and sends typed requests.
type NetworkActions struct {
	Reconfigure    func(iface string) tea.Cmd
	ConfigureIPv4  func(iface string, cfg network.IPv4Config) tea.Cmd
	ResetInterface func(iface string) tea.Cmd
	SetAirplane    func(enabled bool) tea.Cmd
}

// NetworkMsg is dispatched whenever darkd publishes a network snapshot.
type NetworkMsg network.Snapshot

// NetworkActionResultMsg carries the reply for a network mutation.
// On success the included snapshot replaces the cached one; on
// failure the error is shown inline beneath the Interfaces table.
type NetworkActionResultMsg struct {
	Snapshot network.Snapshot
	Err      string
}

func (m *Model) inNetworkContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "network"
}

// triggerNetworkReconfigure asks the active backend to reapply the
// current configuration to the highlighted interface. The "kick this
// thing" verb that triggers a DHCP refresh on systemd-networkd and a
// connection reapply on NetworkManager.
func (m *Model) triggerNetworkAirplaneToggle() tea.Cmd {
	if m.network.SetAirplane == nil || !m.inNetworkContent() || m.state.NetworkBusy {
		return nil
	}
	m.state.NetworkBusy = true
	m.state.NetworkActionError = ""
	return m.network.SetAirplane(!m.state.Network.AirplaneMode)
}

func (m *Model) triggerNetworkReconfigure() tea.Cmd {
	if m.network.Reconfigure == nil || !m.inNetworkContent() || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Name == "" {
		return nil
	}
	m.state.NetworkBusy = true
	m.state.NetworkActionError = ""
	return m.network.Reconfigure(iface.Name)
}

// triggerNetworkUseDHCP commits a DHCP-only IPv4 config to the
// highlighted interface. One keystroke, no dialog — the typical
// "this interface should just get an address from the network"
// path. Surfaces a polkit dialog because the underlying file
// write happens via the privileged helper on systemd-networkd.
func (m *Model) triggerNetworkUseDHCP() tea.Cmd {
	if m.network.ConfigureIPv4 == nil || !m.inNetworkContent() || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Name == "" {
		return nil
	}
	m.state.NetworkBusy = true
	m.state.NetworkActionError = ""
	return m.network.ConfigureIPv4(iface.Name, network.IPv4Config{Mode: "dhcp"})
}

// triggerNetworkEditStatic opens a dialog letting the user fill in a
// static IPv4 configuration. Prefills are sourced from iface.Managed
// when present (the dark-managed config we previously wrote, parsed
// back from the .network file) so the user always edits dark's own
// state rather than the kernel's mixed view.
//
// Routes are NOT in this dialog — they have their own drill-in
// reachable via `t` because the list-of-things UX doesn't fit in a
// fixed-fields dialog cleanly.
//
// On submit, dispatches ConfigureIPv4 with mode="static". Existing
// routes are preserved through the round trip because we copy them
// from iface.Managed into the new IPv4Config.
func (m *Model) triggerNetworkEditStatic() tea.Cmd {
	if m.network.ConfigureIPv4 == nil || !m.inNetworkContent() || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Name == "" {
		return nil
	}
	actions := m.network
	state := m.state
	name := iface.Name

	// Prefer dark's own previously-written config when it exists.
	// Otherwise fall back to the kernel state we have in the
	// snapshot, which is at least a reasonable starting point.
	prefillAddr := ""
	prefillGateway := ""
	prefillDNS := ""
	prefillSearch := ""
	prefillMTU := ""
	prefillV6Mode := ""
	prefillV6Addr := ""
	prefillV6Gateway := ""
	var existingRoutes []network.RouteConfig
	if iface.Managed != nil {
		mc := iface.Managed
		prefillAddr = mc.Address
		prefillGateway = mc.Gateway
		if len(mc.DNS) > 0 {
			prefillDNS = strings.Join(mc.DNS, ", ")
		}
		if len(mc.Search) > 0 {
			prefillSearch = strings.Join(mc.Search, ", ")
		}
		if mc.MTU > 0 {
			prefillMTU = fmt.Sprintf("%d", mc.MTU)
		}
		prefillV6Mode = mc.IPv6Mode
		prefillV6Addr = mc.IPv6Address
		prefillV6Gateway = mc.IPv6Gateway
		existingRoutes = append(existingRoutes, mc.Routes...)
	} else {
		if len(iface.IPv4) > 0 {
			prefillAddr = iface.IPv4[0].CIDR
		}
		if iface.Management != nil && len(iface.Management.DNS) > 0 {
			prefillDNS = strings.Join(iface.Management.DNS, ", ")
		}
		if iface.Management != nil && len(iface.Management.Domains) > 0 {
			prefillSearch = strings.Join(iface.Management.Domains, ", ")
		}
		if iface.MTU > 0 {
			prefillMTU = fmt.Sprintf("%d", iface.MTU)
		}
	}

	m.dialog = NewDialog("Static IPv4 · "+name,
		[]DialogFieldSpec{
			{Key: "address", Label: "IPv4 address (CIDR, e.g. 192.168.1.10/24)", Kind: DialogFieldText, Value: prefillAddr},
			{Key: "gateway", Label: "IPv4 gateway (optional)", Kind: DialogFieldText, Value: prefillGateway},
			{Key: "v6mode", Label: "IPv6 mode (dhcp / static / ra / blank for none)", Kind: DialogFieldText, Value: prefillV6Mode},
			{Key: "v6address", Label: "IPv6 address (CIDR, only for static)", Kind: DialogFieldText, Value: prefillV6Addr},
			{Key: "v6gateway", Label: "IPv6 gateway (optional)", Kind: DialogFieldText, Value: prefillV6Gateway},
			{Key: "dns", Label: "DNS servers (comma-separated, v4 or v6)", Kind: DialogFieldText, Value: prefillDNS},
			{Key: "search", Label: "Search domains (comma-separated)", Kind: DialogFieldText, Value: prefillSearch},
			{Key: "mtu", Label: "MTU bytes (blank = default)", Kind: DialogFieldText, Value: prefillMTU},
		},
		func(result DialogResult) tea.Cmd {
			addr := strings.TrimSpace(result["address"])
			if addr == "" {
				state.NetworkActionError = "IPv4 address is required for static configuration"
				return nil
			}
			var mtu int
			if raw := strings.TrimSpace(result["mtu"]); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n <= 0 {
					state.NetworkActionError = "mtu must be a positive integer"
					return nil
				}
				mtu = n
			}
			v6Mode := strings.ToLower(strings.TrimSpace(result["v6mode"]))
			switch v6Mode {
			case "", "dhcp", "static", "ra":
			default:
				state.NetworkActionError = "ipv6 mode must be dhcp, static, ra, or blank"
				return nil
			}
			v6Addr := strings.TrimSpace(result["v6address"])
			if v6Mode == "static" && v6Addr == "" {
				state.NetworkActionError = "static IPv6 mode requires an address"
				return nil
			}
			cfg := network.IPv4Config{
				Mode:        "static",
				Address:     addr,
				Gateway:     strings.TrimSpace(result["gateway"]),
				DNS:         splitCSV(result["dns"]),
				Search:      splitCSV(result["search"]),
				MTU:         mtu,
				Routes:      existingRoutes,
				IPv6Mode:    v6Mode,
				IPv6Address: v6Addr,
				IPv6Gateway: strings.TrimSpace(result["v6gateway"]),
			}
			state.NetworkBusy = true
			state.NetworkActionError = ""
			return actions.ConfigureIPv4(name, cfg)
		},
	)
	return nil
}

// triggerNetworkRoutesOpen drills into the routes management view for
// the highlighted interface. Returns nil when the interface has no
// dark-managed config — in that case the user is told to set up basic
// IPv4 first (via h or e) before they can add routes.
func (m *Model) triggerNetworkRoutesOpen() tea.Cmd {
	if !m.inNetworkContent() || m.state.NetworkRoutesOpen || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Name == "" {
		return nil
	}
	if iface.Managed == nil {
		m.state.NetworkActionError = "press h or e to configure this interface first, then manage routes"
		return nil
	}
	if !m.state.OpenNetworkRoutes() {
		return nil
	}
	return nil
}

// inNetworkRoutes is true when the user is in the routes drill-in
// for the network section. Used to gate the per-route action keys.
func (m *Model) inNetworkRoutes() bool {
	return m.inNetworkContent() && m.state.NetworkRoutesOpen
}

// triggerNetworkRouteAdd opens a small 3-field dialog (destination,
// gateway, metric) for adding one static route to the highlighted
// interface. On submit, merges with iface.Managed and dispatches a
// full ConfigureIPv4 call with the combined route list.
func (m *Model) triggerNetworkRouteAdd() tea.Cmd {
	if m.network.ConfigureIPv4 == nil || !m.inNetworkRoutes() || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Managed == nil {
		return nil
	}
	actions := m.network
	state := m.state
	name := iface.Name
	existing := *iface.Managed

	m.dialog = NewDialog("Add route · "+name,
		[]DialogFieldSpec{
			{Key: "destination", Label: "Destination (CIDR, e.g. 10.0.0.0/8 or 0.0.0.0/0)", Kind: DialogFieldText},
			{Key: "gateway", Label: "Gateway (optional, blank for on-link)", Kind: DialogFieldText},
			{Key: "metric", Label: "Metric (optional)", Kind: DialogFieldText},
		},
		func(result DialogResult) tea.Cmd {
			dest := strings.TrimSpace(result["destination"])
			if dest == "" {
				state.NetworkActionError = "destination is required"
				return nil
			}
			route := network.RouteConfig{
				Destination: dest,
				Gateway:     strings.TrimSpace(result["gateway"]),
			}
			if raw := strings.TrimSpace(result["metric"]); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 0 {
					state.NetworkActionError = "metric must be a non-negative integer"
					return nil
				}
				route.Metric = n
			}
			merged := existing
			merged.Routes = append(append([]network.RouteConfig{}, existing.Routes...), route)
			state.NetworkBusy = true
			state.NetworkActionError = ""
			return actions.ConfigureIPv4(name, merged)
		},
	)
	return nil
}

// triggerNetworkRouteDelete removes the highlighted route from the
// dark-managed route list and dispatches a full ConfigureIPv4 with
// the remaining routes.
func (m *Model) triggerNetworkRouteDelete() tea.Cmd {
	if m.network.ConfigureIPv4 == nil || !m.inNetworkRoutes() || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Managed == nil {
		return nil
	}
	_, idx, ok := m.state.SelectedNetworkRoute()
	if !ok {
		return nil
	}
	merged := *iface.Managed
	merged.Routes = append(append([]network.RouteConfig{}, iface.Managed.Routes[:idx]...),
		iface.Managed.Routes[idx+1:]...)
	if m.state.NetworkRouteSelected >= len(merged.Routes) && m.state.NetworkRouteSelected > 0 {
		m.state.NetworkRouteSelected--
	}
	m.state.NetworkBusy = true
	m.state.NetworkActionError = ""
	return m.network.ConfigureIPv4(iface.Name, merged)
}

// triggerNetworkReset removes any dark-managed configuration for the
// highlighted interface, returning it to whatever the system would
// have done without dark in the picture. One keystroke, no dialog,
// but the helper still goes through pkexec so the user gets a polkit
// prompt before the file is deleted.
func (m *Model) triggerNetworkReset() tea.Cmd {
	if m.network.ResetInterface == nil || !m.inNetworkContent() || m.state.NetworkBusy {
		return nil
	}
	iface, ok := m.state.SelectedNetworkInterface()
	if !ok || iface.Name == "" {
		return nil
	}
	m.state.NetworkBusy = true
	m.state.NetworkActionError = ""
	return m.network.ResetInterface(iface.Name)
}

// splitCSV trims and splits a comma-separated string, dropping empty
// fragments. Used by the static IP dialog to parse DNS and search
// domain lists from a single text field.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
