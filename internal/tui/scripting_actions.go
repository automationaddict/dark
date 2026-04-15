package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
)

// ScriptingActions is the set of read-only fetch commands the F5
// Scripting tab dispatches on first entry. Phase 1 is browse-only so
// no mutating commands (create/save/delete) exist yet.
type ScriptingActions struct {
	LoadScripts    func() tea.Cmd
	LoadRegistry   func() tea.Cmd
	LoadAPICatalog func() tea.Cmd
	ReadScript     func(name string) tea.Cmd
	SaveScript     func(name, content string) tea.Cmd
	DeleteScript   func(name string) tea.Cmd
}

// ScriptingScriptsMsg carries the response to LoadScripts.
type ScriptingScriptsMsg struct {
	Scripts []core.ScriptEntry
	Err     string
}

// ScriptingRegistryMsg carries the response to LoadRegistry.
type ScriptingRegistryMsg struct {
	Entries []core.LuaRegistryEntry
	Err     string
}

// ScriptingAPICatalogMsg carries the response to LoadAPICatalog.
type ScriptingAPICatalogMsg struct {
	Commands []core.APICommandEntry
	Err      string
}

// ScriptingReadMsg carries the full text of a script ready to open
// in the editor overlay.
type ScriptingReadMsg struct {
	Name    string
	Content string
	Err     string
}

// ScriptingWriteMsg is the shared reply for save and delete. On
// success Scripts carries the refreshed script list so the UI can
// replace its cached snapshot without a second round-trip.
type ScriptingWriteMsg struct {
	Action  string // "save" or "delete"
	Name    string
	Scripts []core.ScriptEntry
	Err     string
}
