package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/keybind"
	"github.com/johnnelson/dark/internal/services/notify"
)

type KeybindActions struct {
	Add    func(b keybind.Binding) tea.Cmd
	Update func(old, new keybind.Binding) tea.Cmd
	Remove func(b keybind.Binding) tea.Cmd
}

type KeybindMsg keybind.Snapshot

type KeybindActionResultMsg struct {
	Snapshot keybind.Snapshot
	Err      string
}

func (m *Model) inKeybindContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabF3 &&
		m.state.ActiveOmarchySection().ID == "keybindings"
}

func (m *Model) triggerKeybindAdd() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 {
		return nil
	}
	keybindRef := m.keybind
	notifierRef := m.notifier
	stateRef := m.state
	m.dialog = NewDialog("Add keybinding", []DialogFieldSpec{
		{Key: "mods", Label: "Modifiers (e.g. SUPER SHIFT)", Value: "SUPER"},
		{Key: "key", Label: "Key"},
		{Key: "desc", Label: "Description"},
		{Key: "dispatcher", Label: "Dispatcher", Value: "exec"},
		{Key: "args", Label: "Arguments"},
	}, func(result DialogResult) tea.Cmd {
		mods := result["mods"]
		key := result["key"]
		desc := result["desc"]
		dispatcher := result["dispatcher"]
		args := result["args"]
		if key == "" || dispatcher == "" {
			return nil
		}

		b := keybind.Binding{
			Mods:       mods,
			Key:        key,
			Desc:       desc,
			Dispatcher: dispatcher,
			Args:       args,
			Source:     keybind.SourceUser,
			BindType:   "bindd",
		}

		conflicts := keybind.DetectConflicts(stateRef.Keybindings.Bindings, mods, key, -1)
		if len(conflicts) > 0 {
			notifyConflicts(notifierRef, mods, key, conflicts)
			// Return a command that opens a confirmation dialog.
			return func() tea.Msg {
				return keybindConflictMsg{binding: b, conflicts: conflicts, action: "add"}
			}
		}

		if keybindRef.Add == nil {
			return nil
		}
		return keybindRef.Add(b)
	})
	return nil
}

func (m *Model) triggerKeybindEdit() tea.Cmd {
	if !m.inKeybindContent() {
		return nil
	}
	old, ok := m.state.SelectedKeybinding()
	if !ok {
		return nil
	}
	keybindRef := m.keybind
	notifierRef := m.notifier
	stateRef := m.state
	// Find the index in the full (unfiltered) bindings list for conflict detection.
	excludeIdx := -1
	for i, b := range stateRef.Keybindings.Bindings {
		if b.Mods == old.Mods && b.Key == old.Key && b.Dispatcher == old.Dispatcher && b.Source == old.Source {
			excludeIdx = i
			break
		}
	}
	m.dialog = NewDialog("Edit keybinding", []DialogFieldSpec{
		{Key: "mods", Label: "Modifiers", Value: old.Mods},
		{Key: "key", Label: "Key", Value: old.Key},
		{Key: "desc", Label: "Description", Value: old.Desc},
		{Key: "dispatcher", Label: "Dispatcher", Value: old.Dispatcher},
		{Key: "args", Label: "Arguments", Value: old.Args},
	}, func(result DialogResult) tea.Cmd {
		mods := result["mods"]
		key := result["key"]
		desc := result["desc"]
		dispatcher := result["dispatcher"]
		args := result["args"]
		if key == "" || dispatcher == "" {
			return nil
		}

		new := keybind.Binding{
			Mods:       mods,
			Key:        key,
			Desc:       desc,
			Dispatcher: dispatcher,
			Args:       args,
			Source:     old.Source,
			Category:   old.Category,
			BindType:   old.BindType,
		}

		conflicts := keybind.DetectConflicts(stateRef.Keybindings.Bindings, mods, key, excludeIdx)
		if len(conflicts) > 0 {
			notifyConflicts(notifierRef, mods, key, conflicts)
			return func() tea.Msg {
				return keybindConflictMsg{binding: new, old: &old, conflicts: conflicts, action: "update"}
			}
		}

		if keybindRef.Update == nil {
			return nil
		}
		return keybindRef.Update(old, new)
	})
	return nil
}

func (m *Model) triggerKeybindRemove() tea.Cmd {
	if !m.inKeybindContent() {
		return nil
	}
	b, ok := m.state.SelectedKeybinding()
	if !ok {
		return nil
	}
	keybindRef := m.keybind
	desc := b.Desc
	if desc == "" {
		desc = b.Mods + " + " + b.Key
	}
	m.dialog = NewDialog("Remove "+desc+"?", nil, func(_ DialogResult) tea.Cmd {
		if keybindRef.Remove == nil {
			return nil
		}
		return keybindRef.Remove(b)
	})
	return nil
}

// keybindConflictMsg is sent when a conflict is detected during add/edit.
// The model handles it by opening a confirmation dialog.
type keybindConflictMsg struct {
	binding   keybind.Binding
	old       *keybind.Binding // nil for add
	conflicts []keybind.Conflict
	action    string // "add" or "update"
}

func (m *Model) handleKeybindConflict(msg keybindConflictMsg) tea.Cmd {
	var descs []string
	for _, c := range msg.conflicts {
		d := c.Existing.Desc
		if d == "" {
			d = c.Existing.Dispatcher
		}
		descs = append(descs, d)
	}
	conflictText := strings.Join(descs, ", ")
	title := fmt.Sprintf("Conflicts with: %s. Override?", conflictText)

	keybindRef := m.keybind
	binding := msg.binding
	old := msg.old
	action := msg.action

	m.dialog = NewDialog(title, nil, func(_ DialogResult) tea.Cmd {
		switch action {
		case "add":
			if keybindRef.Add == nil {
				return nil
			}
			return keybindRef.Add(binding)
		case "update":
			if keybindRef.Update == nil || old == nil {
				return nil
			}
			return keybindRef.Update(*old, binding)
		}
		return nil
	})
	return nil
}

func notifyConflicts(notifier *notify.Notifier, mods, key string, conflicts []keybind.Conflict) {
	if notifier == nil {
		return
	}
	var descs []string
	for _, c := range conflicts {
		d := c.Existing.Desc
		if d == "" {
			d = c.Existing.Dispatcher
		}
		descs = append(descs, d)
	}
	combo := key
	if mods != "" {
		combo = mods + " + " + key
	}
	notifier.Send(notify.Message{
		Summary: "dark · Keybinding Conflict",
		Body:    fmt.Sprintf("%s conflicts with: %s", combo, strings.Join(descs, ", ")),
		Urgency: notify.UrgencyNormal,
		Icon:    "dialog-warning",
	})
}
