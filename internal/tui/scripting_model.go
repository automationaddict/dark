package tui

import (
	"os"
	"path/filepath"
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
	if !m.state.MCPCatalogLoaded && m.scripting.LoadMCPCatalog != nil {
		cmds = append(cmds, m.scripting.LoadMCPCatalog())
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// triggerScriptingEnter handles Enter inside the F5 Scripting tab.
// First Enter moves focus into the currently-selected group's inner
// sub-nav. A second Enter inside the Scripts group suspends the TUI
// and hands off to $EDITOR — either on a freshly-created empty file
// (new script) or on the highlighted script's path directly. On
// editor exit darkd reloads every user script so hooks re-register
// without a daemon restart.
func (m *Model) triggerScriptingEnter() tea.Cmd {
	if !m.state.ScriptingContentFocused {
		m.state.ScriptingContentFocused = true
		return nil
	}
	if m.state.ScriptingSelection.Kind != core.SelKindScripts {
		return nil
	}
	if m.state.ScriptsInnerIdx == 0 {
		m.dialog = NewDialog("New Lua script", []DialogFieldSpec{
			{Key: "name", Label: "Filename", Kind: DialogFieldText, Value: ".lua"},
		}, func(result DialogResult) tea.Cmd {
			name := normalizeScriptName(result["name"])
			if name == "" {
				return nil
			}
			path, err := createEmptyScriptFile(name)
			if err != nil {
				m.notifyError("Scripting", err.Error())
				return nil
			}
			return editExistingFile(editKindScript, name, path)
		})
		return nil
	}
	sc, ok := m.selectedScript()
	if !ok {
		return nil
	}
	return editExistingFile(editKindScript, sc.Name, sc.Path)
}

// handlePageKey routes PgUp/PgDn. On F5 with a script selected it
// scrolls the preview window; elsewhere it's a no-op (for now).
func (m *Model) handlePageKey(direction int) {
	if m.state.ActiveTab != core.TabF5 {
		return
	}
	if m.state.ScriptingSelection.Kind != core.SelKindScripts {
		return
	}
	if _, ok := m.state.SelectedScriptIdx(); !ok {
		return
	}
	m.state.ScrollScriptPreview(direction * 10)
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

// selectedScript returns the script currently pointed at by the
// Scripts inner sub-nav. Returns false when the pointer is on the
// `+ New script` row, on another group entirely, or out of range.
func (m *Model) selectedScript() (core.ScriptEntry, bool) {
	if m.state.ScriptingSelection.Kind != core.SelKindScripts {
		return core.ScriptEntry{}, false
	}
	idx, ok := m.state.SelectedScriptIdx()
	if !ok {
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

// createEmptyScriptFile writes a zero-byte file at the given script
// name inside the user scripts directory so $EDITOR can open the
// real path directly (no temp-file dance). Returns the absolute
// path. Refuses to overwrite if something already exists — the name
// dialog should have caught the conflict, but it's defensive.
func createEmptyScriptFile(name string) (string, error) {
	dir := clientUserScriptsDir()
	if dir == "" {
		return "", os.ErrNotExist
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	f.Close()
	return path, nil
}

// clientUserScriptsDir mirrors scripting.UserScriptsDir on the
// client side — both dark and darkd run as the same user and share
// XDG resolution, so duplicating this short helper is cheaper than
// introducing a client/daemon import edge.
func clientUserScriptsDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "dark", "scripts")
}
