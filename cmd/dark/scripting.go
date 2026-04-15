package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/scripting"
	"github.com/automationaddict/dark/internal/tui"
)

// newScriptingActions wires the three F5 Scripting fetch commands to
// NATS request/reply round-trips against darkd. All three are
// read-only in phase 1 — the UI calls them once on first entry into
// the tab and the daemon responds with the current snapshot.
func newScriptingActions(nc *nats.Conn) tui.ScriptingActions {
	return tui.ScriptingActions{
		LoadScripts: func() tea.Cmd {
			return func() tea.Msg {
				return scriptingListRequest(nc)
			}
		},
		LoadRegistry: func() tea.Cmd {
			return func() tea.Msg {
				return scriptingRegistryRequest(nc)
			}
		},
		LoadAPICatalog: func() tea.Cmd {
			return func() tea.Msg {
				return scriptingAPICatalogRequest(nc)
			}
		},
		LoadMCPCatalog: func() tea.Cmd {
			return func() tea.Msg {
				return scriptingMCPCatalogRequest(nc)
			}
		},
		SaveScript: func(name, content string) tea.Cmd {
			return func() tea.Msg {
				return scriptingWriteRequest(nc, "save", name, content)
			}
		},
		DeleteScript: func(name string) tea.Cmd {
			return func() tea.Msg {
				return scriptingWriteRequest(nc, "delete", name, "")
			}
		},
		ReloadScripts: func() tea.Cmd {
			return func() tea.Msg {
				return scriptingWriteRequest(nc, "reload", "", "")
			}
		},
	}
}

func scriptingWriteRequest(nc *nats.Conn, action, name, content string) tui.ScriptingWriteMsg {
	var subject string
	payloadMap := map[string]string{"name": name}
	switch action {
	case "save":
		subject = bus.SubjectScriptingSaveCmd
		payloadMap["content"] = content
	case "delete":
		subject = bus.SubjectScriptingDeleteCmd
	case "reload":
		subject = bus.SubjectScriptingReloadCmd
		payloadMap = nil
	default:
		return tui.ScriptingWriteMsg{Action: action, Name: name, Err: "unknown action: " + action}
	}
	var payload []byte
	if payloadMap != nil {
		payload, _ = json.Marshal(payloadMap)
	}
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.ScriptingWriteMsg{Action: action, Name: name, Err: err.Error()}
	}
	var resp struct {
		Name    string                 `json:"name"`
		Scripts []scripting.ScriptFile `json:"scripts,omitempty"`
		Error   string                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.ScriptingWriteMsg{Action: action, Name: name, Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.ScriptingWriteMsg{Action: action, Name: name, Err: resp.Error}
	}
	out := make([]core.ScriptEntry, 0, len(resp.Scripts))
	for _, s := range resp.Scripts {
		out = append(out, core.ScriptEntry{
			Name:      s.Name,
			Path:      s.Path,
			Source:    s.Source,
			SizeBytes: s.SizeBytes,
			ModTime:   s.ModTime,
			Preview:   s.Preview,
		})
	}
	return tui.ScriptingWriteMsg{Action: action, Name: resp.Name, Scripts: out}
}

func scriptingListRequest(nc *nats.Conn) tui.ScriptingScriptsMsg {
	reply, err := nc.Request(bus.SubjectScriptingListCmd, nil, core.TimeoutNormal)
	if err != nil {
		return tui.ScriptingScriptsMsg{Err: err.Error()}
	}
	var resp struct {
		Scripts []scripting.ScriptFile `json:"scripts"`
		Error   string                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.ScriptingScriptsMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.ScriptingScriptsMsg{Err: resp.Error}
	}
	out := make([]core.ScriptEntry, 0, len(resp.Scripts))
	for _, s := range resp.Scripts {
		out = append(out, core.ScriptEntry{
			Name:      s.Name,
			Path:      s.Path,
			Source:    s.Source,
			SizeBytes: s.SizeBytes,
			ModTime:   s.ModTime,
			Preview:   s.Preview,
		})
	}
	return tui.ScriptingScriptsMsg{Scripts: out}
}

func scriptingRegistryRequest(nc *nats.Conn) tui.ScriptingRegistryMsg {
	reply, err := nc.Request(bus.SubjectScriptingRegistryCmd, nil, core.TimeoutNormal)
	if err != nil {
		return tui.ScriptingRegistryMsg{Err: err.Error()}
	}
	var resp struct {
		Entries []scripting.RegistryEntry `json:"entries"`
		Error   string                    `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.ScriptingRegistryMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.ScriptingRegistryMsg{Err: resp.Error}
	}
	out := make([]core.LuaRegistryEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		fields := make([]core.CommandField, 0, len(e.Fields))
		for _, f := range e.Fields {
			fields = append(fields, core.CommandField{
				Name:     f.Name,
				Type:     f.Type,
				Required: f.Required,
				Desc:     f.Desc,
			})
		}
		out = append(out, core.LuaRegistryEntry{
			Kind:    e.Kind,
			Name:    e.Name,
			Args:    e.Args,
			Summary: e.Summary,
			Subject: e.Subject,
			Fields:  fields,
		})
	}
	return tui.ScriptingRegistryMsg{Entries: out}
}

func scriptingMCPCatalogRequest(nc *nats.Conn) tui.ScriptingMCPCatalogMsg {
	reply, err := nc.Request(bus.SubjectScriptingMCPCatalogCmd, nil, core.TimeoutNormal)
	if err != nil {
		return tui.ScriptingMCPCatalogMsg{Err: err.Error()}
	}
	// Decode into a shape that mirrors internal/mcp's JSON tags
	// without importing that package here — same pattern the other
	// scripting requests use to stay decoupled from daemon types.
	var resp struct {
		Tools []struct {
			Name    string            `json:"name"`
			Subject string            `json:"subject"`
			Domain  string            `json:"domain"`
			Verb    string            `json:"verb"`
			Summary string            `json:"summary,omitempty"`
			Fields  []bus.CommandField `json:"fields,omitempty"`
		} `json:"tools"`
		Resources []struct {
			URI     string `json:"uri"`
			Name    string `json:"name"`
			Subject string `json:"subject"`
			Summary string `json:"summary,omitempty"`
		} `json:"resources"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.ScriptingMCPCatalogMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.ScriptingMCPCatalogMsg{Err: resp.Error}
	}
	tools := make([]core.MCPToolEntry, 0, len(resp.Tools))
	for _, t := range resp.Tools {
		fields := make([]core.CommandField, 0, len(t.Fields))
		for _, f := range t.Fields {
			fields = append(fields, core.CommandField{
				Name:     f.Name,
				Type:     f.Type,
				Required: f.Required,
				Desc:     f.Desc,
			})
		}
		tools = append(tools, core.MCPToolEntry{
			Name:    t.Name,
			Subject: t.Subject,
			Domain:  t.Domain,
			Verb:    t.Verb,
			Summary: t.Summary,
			Fields:  fields,
		})
	}
	resources := make([]core.MCPResourceEntry, 0, len(resp.Resources))
	for _, r := range resp.Resources {
		resources = append(resources, core.MCPResourceEntry{
			URI:     r.URI,
			Name:    r.Name,
			Subject: r.Subject,
			Summary: r.Summary,
		})
	}
	return tui.ScriptingMCPCatalogMsg{Tools: tools, Resources: resources}
}

func scriptingAPICatalogRequest(nc *nats.Conn) tui.ScriptingAPICatalogMsg {
	reply, err := nc.Request(bus.SubjectScriptingAPICatalogCmd, nil, core.TimeoutNormal)
	if err != nil {
		return tui.ScriptingAPICatalogMsg{Err: err.Error()}
	}
	var resp struct {
		Commands []bus.APICommandEntry `json:"commands"`
		Error    string                `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.ScriptingAPICatalogMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.ScriptingAPICatalogMsg{Err: resp.Error}
	}
	out := make([]core.APICommandEntry, 0, len(resp.Commands))
	for _, c := range resp.Commands {
		fields := make([]core.CommandField, 0, len(c.Fields))
		for _, f := range c.Fields {
			fields = append(fields, core.CommandField{
				Name:     f.Name,
				Type:     f.Type,
				Required: f.Required,
				Desc:     f.Desc,
			})
		}
		out = append(out, core.APICommandEntry{
			Subject: c.Subject,
			Domain:  c.Domain,
			Verb:    c.Verb,
			Summary: c.Summary,
			Fields:  fields,
		})
	}
	return tui.ScriptingAPICatalogMsg{Commands: out}
}
