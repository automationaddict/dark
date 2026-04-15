package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
)

// loadScriptingIfNeeded fires the three F5 fetch commands the first
// time the user enters the Scripting tab. Subsequent entries are
// no-ops — the cached state stays until the session ends.
func (m *Model) loadScriptingIfNeeded() tea.Cmd {
	var cmds []tea.Cmd
	if !m.state.ScriptsLoaded && m.scripting.LoadScripts != nil {
		cmds = append(cmds, m.scripting.LoadScripts())
	}
	if !m.state.LuaRegistryLoaded && m.scripting.LoadRegistry != nil {
		cmds = append(cmds, m.scripting.LoadRegistry())
	}
	if !m.state.APICommandsLoaded && m.scripting.LoadAPICatalog != nil {
		cmds = append(cmds, m.scripting.LoadAPICatalog())
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// triggerScriptingEnter handles Enter inside the F5 Scripting tab.
// The first Enter on a reference row (MCP / Lua / API) moves focus
// into the inner sub-nav so arrow keys drive the command list. On a
// script row it fetches the file contents so the reply can open the
// editor. On the "+ New script" row it opens a name-entry dialog.
func (m *Model) triggerScriptingEnter() tea.Cmd {
	if m.state.ContentFocused {
		// Second Enter inside the content pane is unused for now.
		return nil
	}
	switch m.state.ScriptingSelection.Kind {
	case core.SelKindNewScript:
		m.dialog = NewDialog("New Lua script", []DialogFieldSpec{
			{Key: "name", Label: "Filename", Kind: DialogFieldText, Value: ".lua"},
		}, func(result DialogResult) tea.Cmd {
			name := normalizeScriptName(result["name"])
			if name == "" {
				return nil
			}
			m.openScriptEditor(name, "")
			return nil
		})
		return nil
	case core.SelKindScript:
		sc, ok := m.selectedScript()
		if !ok || m.scripting.ReadScript == nil {
			return nil
		}
		return m.scripting.ReadScript(sc.Name)
	case core.SelKindMCP, core.SelKindLua, core.SelKindAPI:
		m.state.ContentFocused = true
		return nil
	}
	return nil
}

// triggerScriptingDelete fires on `d` while a user script row is
// highlighted. Opens a confirmation dialog and dispatches DeleteScript
// on accept. No-op when the current selection isn't a script.
func (m *Model) triggerScriptingDelete() tea.Cmd {
	sc, ok := m.selectedScript()
	if !ok || m.scripting.DeleteScript == nil {
		return nil
	}
	name := sc.Name
	m.dialog = NewDialog("Delete "+name+"?", nil, func(_ DialogResult) tea.Cmd {
		return m.scripting.DeleteScript(name)
	})
	return nil
}

// openScriptEditor installs a Lua-syntax editor overlay with the
// given initial content. Ctrl+S dispatches a SaveScript action; Esc
// discards without saving.
func (m *Model) openScriptEditor(name, content string) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 40
	}
	saveFn := m.scripting.SaveScript
	m.editor = NewEditorWithLanguage(name, LangLua, content, width, height,
		func(final string) tea.Cmd {
			if saveFn == nil {
				return nil
			}
			return saveFn(name, final)
		})
}

// selectedScript returns the script currently pointed at by the
// sidebar. Returns false when the selection isn't a script row or
// the index has drifted out of range.
func (m *Model) selectedScript() (core.ScriptEntry, bool) {
	if m.state.ScriptingSelection.Kind != core.SelKindScript {
		return core.ScriptEntry{}, false
	}
	idx := m.state.ScriptingSelection.Index
	if idx < 0 || idx >= len(m.state.Scripts) {
		return core.ScriptEntry{}, false
	}
	return m.state.Scripts[idx], true
}

// normalizeScriptName trims whitespace and appends `.lua` when the
// user typed a bare basename. Empty results propagate as cancel.
func normalizeScriptName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" || name == ".lua" {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(name), ".lua") {
		name += ".lua"
	}
	return name
}
