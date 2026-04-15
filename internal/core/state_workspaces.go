package core

import "github.com/automationaddict/dark/internal/services/workspaces"

// WorkspacesSection describes one sub-section visible in the
// Workspaces inner sidebar.
type WorkspacesSection struct {
	ID    string
	Icon  string
	Label string
}

func WorkspacesSections() []WorkspacesSection {
	return []WorkspacesSection{
		{"overview", "󰕮", "Overview"},
		{"layout", "󱇐", "Layout"},
		{"behavior", "󰒓", "Behavior"},
	}
}

func (s *State) ActiveWorkspacesSection() WorkspacesSection {
	secs := WorkspacesSections()
	if s.WorkspacesSectionIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.WorkspacesSectionIdx]
}

func (s *State) MoveWorkspacesSection(delta int) {
	n := len(WorkspacesSections())
	if n == 0 {
		return
	}
	s.WorkspacesSectionIdx = (s.WorkspacesSectionIdx + delta + n) % n
	s.WorkspacesContentIdx = 0
}

// SetWorkspaces replaces the cached workspaces snapshot and clears
// the busy flag. Clamps the content selection to the new length so
// a shrinking list doesn't leave the cursor pointing at a deleted
// row.
func (s *State) SetWorkspaces(snap workspaces.Snapshot) {
	s.Workspaces = snap
	s.WorkspacesLoaded = true
	s.WorkspacesBusy = false
	if s.WorkspacesContentIdx >= len(snap.Workspaces) && len(snap.Workspaces) > 0 {
		s.WorkspacesContentIdx = len(snap.Workspaces) - 1
	}
	if len(snap.Workspaces) == 0 {
		s.WorkspacesContentIdx = 0
	}
}

// SelectedWorkspace returns the highlighted workspace row on the
// Overview sub-section. The bool is false when the list is empty.
func (s *State) SelectedWorkspace() (workspaces.Workspace, bool) {
	ws := s.Workspaces.Workspaces
	if len(ws) == 0 {
		return workspaces.Workspace{}, false
	}
	if s.WorkspacesContentIdx >= len(ws) {
		s.WorkspacesContentIdx = 0
	}
	return ws[s.WorkspacesContentIdx], true
}

// MoveWorkspaceSelection moves the highlighted row within the
// active sub-section's list by delta. No-op when the active
// sub-section doesn't have a list (Layout / Behavior show detail
// rows, not selectable entries).
func (s *State) MoveWorkspaceSelection(delta int) {
	if s.ActiveWorkspacesSection().ID != "overview" {
		return
	}
	n := len(s.Workspaces.Workspaces)
	if n == 0 {
		return
	}
	s.WorkspacesContentIdx = (s.WorkspacesContentIdx + delta + n) % n
}
