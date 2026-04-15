package core

import "time"

// ScriptingSelectionKind names an entry in the outer F5 sidebar. The
// outer sidebar is deliberately short (one row per script plus three
// reference groups and a "new" affordance) so it fits without
// scrolling. Every individual MCP / Lua / API command lives in the
// content pane's inner sub-nav, not out here — matching how F1 wifi
// stacks adapters / networks / known under a section-level outer
// sidebar.
type ScriptingSelectionKind int

const (
	SelKindNewScript ScriptingSelectionKind = iota
	SelKindScript
	SelKindMCP
	SelKindLua
	SelKindAPI
)

// ScriptingSelection points at the currently highlighted outer row.
// Index is only meaningful for SelKindScript (index into s.Scripts).
type ScriptingSelection struct {
	Kind  ScriptingSelectionKind
	Index int
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

// MCPEntryCount is the number of rows shown in the MCP inner sub-nav.
// Phase 1 ships a single "coming soon" row.
func (s *State) MCPEntryCount() int { return 1 }

// ScriptingOuterLen is the number of selectable rows in the outer F5
// sidebar: + New script, one per user script, and the three reference
// groups (MCP / Lua / API).
func (s *State) ScriptingOuterLen() int {
	return 1 + len(s.Scripts) + 3
}

// ScriptingOuterAt resolves a flat outer-sidebar index into a
// selection. Out-of-range values fall back to the "+ New script" row.
func (s *State) ScriptingOuterAt(idx int) ScriptingSelection {
	if idx <= 0 {
		return ScriptingSelection{Kind: SelKindNewScript}
	}
	if idx < 1+len(s.Scripts) {
		return ScriptingSelection{Kind: SelKindScript, Index: idx - 1}
	}
	switch idx - (1 + len(s.Scripts)) {
	case 0:
		return ScriptingSelection{Kind: SelKindMCP}
	case 1:
		return ScriptingSelection{Kind: SelKindLua}
	case 2:
		return ScriptingSelection{Kind: SelKindAPI}
	}
	return ScriptingSelection{Kind: SelKindAPI}
}

// ScriptingOuterIndex maps the current selection back to its outer
// sidebar row number.
func (s *State) ScriptingOuterIndex() int {
	sel := s.ScriptingSelection
	switch sel.Kind {
	case SelKindNewScript:
		return 0
	case SelKindScript:
		return 1 + clamp(sel.Index, 0, maxInt(len(s.Scripts)-1, 0))
	case SelKindMCP:
		return 1 + len(s.Scripts)
	case SelKindLua:
		return 2 + len(s.Scripts)
	case SelKindAPI:
		return 3 + len(s.Scripts)
	}
	return 0
}

// MoveScriptingOuter walks the outer sidebar, clamping to its bounds.
func (s *State) MoveScriptingOuter(delta int) {
	n := s.ScriptingOuterLen()
	if n == 0 {
		return
	}
	idx := s.ScriptingOuterIndex() + delta
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	s.ScriptingSelection = s.ScriptingOuterAt(idx)
}

// MoveScriptingInner walks the inner sub-nav inside the content pane
// for the currently-selected reference group. Script / NewScript
// selections have no inner list and the call is a no-op.
func (s *State) MoveScriptingInner(delta int) {
	switch s.ScriptingSelection.Kind {
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

// SetScripts replaces the cached script list and keeps the outer
// selection valid if the list shrank.
func (s *State) SetScripts(list []ScriptEntry) {
	s.Scripts = list
	s.ScriptsLoaded = true
	if s.ScriptingSelection.Kind == SelKindScript {
		if s.ScriptingSelection.Index >= len(list) {
			if len(list) == 0 {
				s.ScriptingSelection = ScriptingSelection{Kind: SelKindNewScript}
			} else {
				s.ScriptingSelection.Index = len(list) - 1
			}
		}
	}
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
