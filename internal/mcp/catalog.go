// Package mcp exposes darkd's bus command surface to Model Context
// Protocol clients. It turns every `dark.cmd.*` subject from the bus
// catalog into an MCP tool and a curated list of snapshot subjects
// into MCP resources, so an LLM can drive the daemon through the
// same vocabulary F5 Scripting surfaces to Lua.
//
// The package has two responsibilities:
//   - Catalog: pure derivation from bus.APICommandCatalog into
//     ToolEntry / ResourceEntry slices. Callable from any process;
//     no bus connection required. Used by the F5 MCP tab (via a
//     darkd handler) to render the same per-entry docs pattern the
//     Lua and API tabs already use.
//   - Server: live MCP JSON-RPC server over stdio, backed by a
//     real NATS connection. Used by `dark mcp` so Claude Desktop
//     and other MCP hosts can talk to a running darkd.
package mcp

import (
	"strings"

	"github.com/automationaddict/dark/internal/bus"
)

// ToolEntry describes one MCP tool exposed by the server. Tools map
// 1:1 to bus command subjects — every `dark.cmd.<domain>.<verb>`
// becomes a tool named `<domain>_<verb>` (dots are not valid in MCP
// tool names). Fields carries the payload schema so the F5 MCP tab
// can render a parameter table the same way the API tab does.
type ToolEntry struct {
	Name    string             `json:"name"`
	Subject string             `json:"subject"`
	Domain  string             `json:"domain"`
	Verb    string             `json:"verb"`
	Summary string             `json:"summary,omitempty"`
	Fields  []bus.CommandField `json:"fields,omitempty"`
}

// ResourceEntry describes one MCP resource. Resources are read-only
// views of daemon state — every entry is backed by a zero-payload
// snapshot command that returns the current JSON snapshot. The URI
// uses the `dark://snapshots/<name>` scheme so LLM hosts can display
// a stable identifier in their UI.
type ResourceEntry struct {
	URI     string `json:"uri"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Summary string `json:"summary,omitempty"`
}

// resourceBindings hand-picks the snapshot commands that make sense as
// MCP resources. Every entry must point at a zero-field command whose
// reply is a JSON snapshot. Hand-curated rather than auto-detected so
// we don't accidentally expose a mutating command as a read resource.
var resourceBindings = []struct {
	name    string
	subject string
}{
	{"system", bus.SubjectSystemInfoCmd},
	{"wifi", bus.SubjectWifiAdaptersCmd},
	{"bluetooth", bus.SubjectBluetoothAdaptersCmd},
	{"audio", bus.SubjectAudioDevicesCmd},
	{"display", bus.SubjectDisplayMonitorsCmd},
	{"network", bus.SubjectNetworkSnapshotCmd},
	{"datetime", bus.SubjectDateTimeSnapshotCmd},
	{"notifications", bus.SubjectNotifySnapshotCmd},
	{"input", bus.SubjectInputSnapshotCmd},
	{"power", bus.SubjectPowerSnapshotCmd},
	{"keybindings", bus.SubjectKeybindSnapshotCmd},
	{"users", bus.SubjectUsersSnapshotCmd},
	{"privacy", bus.SubjectPrivacySnapshotCmd},
	{"appearance", bus.SubjectAppearanceSnapshotCmd},
	{"firmware", bus.SubjectFirmwareSnapshotCmd},
	{"limine", bus.SubjectLimineSnapshotCmd},
	{"screensaver", bus.SubjectScreensaverSnapshotCmd},
	{"topbar", bus.SubjectTopBarSnapshotCmd},
	{"workspaces", bus.SubjectWorkspacesSnapshotCmd},
	{"update", bus.SubjectUpdateSnapshotCmd},
}

// Tools returns one ToolEntry per bus command subject, sorted by
// domain then verb. Derived from bus.APICommandCatalog so it never
// drifts from the real command surface.
func Tools() []ToolEntry {
	cat := bus.APICommandCatalog()
	out := make([]ToolEntry, 0, len(cat))
	for _, c := range cat {
		if c.Domain == "" || c.Verb == "" {
			continue
		}
		out = append(out, ToolEntry{
			Name:    toolName(c.Domain, c.Verb),
			Subject: c.Subject,
			Domain:  c.Domain,
			Verb:    c.Verb,
			Summary: c.Summary,
			Fields:  c.Fields,
		})
	}
	return out
}

// Resources returns the curated snapshot resource list, sorted by
// name. The subject field points at the zero-payload bus command the
// server should request to fetch the current snapshot.
func Resources() []ResourceEntry {
	out := make([]ResourceEntry, 0, len(resourceBindings))
	summaries := resourceSummaries()
	for _, b := range resourceBindings {
		out = append(out, ResourceEntry{
			URI:     "dark://snapshots/" + b.name,
			Name:    b.name,
			Subject: b.subject,
			Summary: summaries[b.subject],
		})
	}
	return out
}

// toolName converts a bus subject's domain/verb into an MCP tool
// name. MCP tool names allow [a-z0-9_-]; our subjects already use
// lowercase identifiers, so we just join with an underscore.
func toolName(domain, verb string) string {
	verb = strings.ReplaceAll(verb, ".", "_")
	return domain + "_" + verb
}

// resourceSummaries pulls one-liners from the bus catalog so the F5
// MCP tab and MCP client UIs show the same description for a snapshot
// whether the user sees it as a tool or a resource.
func resourceSummaries() map[string]string {
	m := map[string]string{}
	for _, c := range bus.APICommandCatalog() {
		m[c.Subject] = c.Summary
	}
	return m
}
