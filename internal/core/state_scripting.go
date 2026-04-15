package core

import "time"

// ScriptingSelectionKind names a top-level row in the outer F5
// sidebar. The sidebar is deliberately short — one row per group —
// and each group's contents (script files, MCP tools, Lua symbols,
// API commands) live in the content pane's inner sub-nav. This
// mirrors how F1 wifi keeps its section selector in the outer
// sidebar and stacks the sub-tables under it.
type ScriptingSelectionKind int

const (
	SelKindScripts ScriptingSelectionKind = iota
	SelKindMCP
	SelKindLua
	SelKindAPI
)

// ScriptingSelection points at the currently highlighted outer row.
type ScriptingSelection struct {
	Kind ScriptingSelectionKind
}

// ScriptEntry is a single user-editable Lua script discovered under
// the user scripts directory.
type ScriptEntry struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Source    string    `json:"source"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
	Preview   string    `json:"preview,omitempty"`
}

// CommandField describes one payload parameter accepted by a bus
// command. Mirrors bus.CommandField on the client side so the tui
// package doesn't have to import bus.
type CommandField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Desc     string `json:"desc"`
}

// LuaRegistryEntry describes one Lua symbol exposed to user scripts.
type LuaRegistryEntry struct {
	Kind    string         `json:"kind"`
	Name    string         `json:"name"`
	Args    string         `json:"args"`
	Summary string         `json:"summary"`
	Subject string         `json:"subject,omitempty"`
	Fields  []CommandField `json:"fields,omitempty"`
}

// APICommandEntry is one enumerated NATS command subject.
type APICommandEntry struct {
	Subject string         `json:"subject"`
	Domain  string         `json:"domain"`
	Verb    string         `json:"verb"`
	Summary string         `json:"summary,omitempty"`
	Fields  []CommandField `json:"fields,omitempty"`
}

// MCPToolEntry mirrors mcp.ToolEntry for the TUI so this package has
// no dependency on internal/mcp (which pulls in mcp-go). One entry
// per MCP tool exposed by `dark mcp`. Fields carries the payload
// schema so the F5 MCP detail pane can render a real parameter
// table the same way the Lua and API tabs do.
type MCPToolEntry struct {
	Name    string         `json:"name"`
	Subject string         `json:"subject"`
	Domain  string         `json:"domain"`
	Verb    string         `json:"verb"`
	Summary string         `json:"summary,omitempty"`
	Fields  []CommandField `json:"fields,omitempty"`
}

// MCPResourceEntry mirrors mcp.ResourceEntry for the TUI. One entry
// per read-only snapshot resource exposed by `dark mcp`.
type MCPResourceEntry struct {
	URI     string `json:"uri"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	Summary string `json:"summary,omitempty"`
}

// MCPEntryCount is the number of navigable rows in the MCP inner
// sub-nav. Equals total tools + total resources; section headers in
// the sidebar render but don't count toward navigation.
func (s *State) MCPEntryCount() int {
	return len(s.MCPTools) + len(s.MCPResources)
}

// SetMCPCatalog replaces the cached MCP tool and resource lists and
// keeps the inner pointer valid if either shrank.
func (s *State) SetMCPCatalog(tools []MCPToolEntry, resources []MCPResourceEntry) {
	s.MCPTools = tools
	s.MCPResources = resources
	s.MCPCatalogLoaded = true
	if s.MCPInnerIdx >= s.MCPEntryCount() {
		s.MCPInnerIdx = 0
	}
}

// SelectedMCPTool returns the tool the inner sub-nav is currently
// pointing at. The boolean is false when the selection falls in the
// resources range or is out of bounds.
func (s *State) SelectedMCPTool() (MCPToolEntry, bool) {
	if s.MCPInnerIdx < 0 || s.MCPInnerIdx >= len(s.MCPTools) {
		return MCPToolEntry{}, false
	}
	return s.MCPTools[s.MCPInnerIdx], true
}

// SelectedMCPResource returns the resource the inner sub-nav is
// currently pointing at. The boolean is false when the selection
// falls in the tools range or is out of bounds.
func (s *State) SelectedMCPResource() (MCPResourceEntry, bool) {
	idx := s.MCPInnerIdx - len(s.MCPTools)
	if idx < 0 || idx >= len(s.MCPResources) {
		return MCPResourceEntry{}, false
	}
	return s.MCPResources[idx], true
}

// ScriptsInnerLen is the number of selectable rows in the Scripts
// group's inner sub-nav: one `+ New script` affordance followed by
// one row per discovered user script.
func (s *State) ScriptsInnerLen() int {
	return 1 + len(s.Scripts)
}

// ScriptingOuterLen is the number of selectable rows in the outer F5
// sidebar (Scripts / MCP / Lua / API).
func (s *State) ScriptingOuterLen() int { return 4 }

// ScriptingOuterAt resolves a flat outer-sidebar index into a
// selection. Out-of-range values clamp to the ends of the list.
func (s *State) ScriptingOuterAt(idx int) ScriptingSelection {
	switch idx {
	case 0:
		return ScriptingSelection{Kind: SelKindScripts}
	case 1:
		return ScriptingSelection{Kind: SelKindMCP}
	case 2:
		return ScriptingSelection{Kind: SelKindLua}
	case 3:
		return ScriptingSelection{Kind: SelKindAPI}
	}
	if idx < 0 {
		return ScriptingSelection{Kind: SelKindScripts}
	}
	return ScriptingSelection{Kind: SelKindAPI}
}

// ScriptingOuterIndex maps the current selection back to its outer
// sidebar row number.
func (s *State) ScriptingOuterIndex() int {
	switch s.ScriptingSelection.Kind {
	case SelKindScripts:
		return 0
	case SelKindMCP:
		return 1
	case SelKindLua:
		return 2
	case SelKindAPI:
		return 3
	}
	return 0
}

// MoveScriptingOuter walks the outer sidebar, clamping to its bounds.
func (s *State) MoveScriptingOuter(delta int) {
	idx := s.ScriptingOuterIndex() + delta
	if idx < 0 {
		idx = 0
	}
	if idx >= s.ScriptingOuterLen() {
		idx = s.ScriptingOuterLen() - 1
	}
	s.ScriptingSelection = s.ScriptingOuterAt(idx)
}

// MoveScriptingInner walks the inner sub-nav inside the content pane
// for the currently-selected group.
func (s *State) MoveScriptingInner(delta int) {
	switch s.ScriptingSelection.Kind {
	case SelKindScripts:
		n := s.ScriptsInnerLen()
		prev := s.ScriptsInnerIdx
		s.ScriptsInnerIdx = clamp(s.ScriptsInnerIdx+delta, 0, maxInt(n-1, 0))
		if prev != s.ScriptsInnerIdx {
			s.ScriptPreviewScroll = 0
		}
	case SelKindMCP:
		n := s.MCPEntryCount()
		s.MCPInnerIdx = clamp(s.MCPInnerIdx+delta, 0, maxInt(n-1, 0))
	case SelKindLua:
		n := len(s.LuaRegistry)
		s.LuaInnerIdx = clamp(s.LuaInnerIdx+delta, 0, maxInt(n-1, 0))
	case SelKindAPI:
		n := len(s.APICommands)
		s.APIInnerIdx = clamp(s.APIInnerIdx+delta, 0, maxInt(n-1, 0))
	}
}

// SelectedScriptIdx returns the index into s.Scripts the inner
// sub-nav is currently pointing at. The second return is false when
// the pointer is on the `+ New script` row or out of range.
func (s *State) SelectedScriptIdx() (int, bool) {
	if s.ScriptsInnerIdx <= 0 {
		return 0, false
	}
	idx := s.ScriptsInnerIdx - 1
	if idx >= len(s.Scripts) {
		return 0, false
	}
	return idx, true
}

// SetScripts replaces the cached script list and keeps the inner
// pointer valid if the list shrank. Preview scroll resets whenever
// the list changes so a freshly-written file starts at the top.
func (s *State) SetScripts(list []ScriptEntry) {
	s.Scripts = list
	s.ScriptsLoaded = true
	max := s.ScriptsInnerLen() - 1
	if max < 0 {
		max = 0
	}
	if s.ScriptsInnerIdx > max {
		s.ScriptsInnerIdx = max
	}
	s.ScriptPreviewScroll = 0
}

// ScrollScriptPreview moves the script preview window by delta
// lines, clamped to [0, ScriptPreviewScrollMax].
func (s *State) ScrollScriptPreview(delta int) {
	s.ScriptPreviewScroll = clamp(
		s.ScriptPreviewScroll+delta,
		0,
		s.ScriptPreviewScrollMax,
	)
}

// SetLuaRegistry replaces the cached Lua registry list.
func (s *State) SetLuaRegistry(list []LuaRegistryEntry) {
	s.LuaRegistry = list
	s.LuaRegistryLoaded = true
	if s.LuaInnerIdx >= len(list) {
		s.LuaInnerIdx = 0
	}
}

// SetAPICommands replaces the cached API command catalog.
func (s *State) SetAPICommands(list []APICommandEntry) {
	s.APICommands = list
	s.APICommandsLoaded = true
	if s.APIInnerIdx >= len(list) {
		s.APIInnerIdx = 0
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
