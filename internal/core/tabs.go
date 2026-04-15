package core

import "strings"

type TabID int

const (
	TabSettings TabID = iota
	TabF2
	TabF3
	TabF4
	TabF5
	TabF6
	TabF7
	TabF8
	TabF9
	TabF10
	TabF11
	TabF12
)

type Tab struct {
	ID     TabID
	Key    string
	Title  string
	Hidden bool
}

// AllTabs returns every tab the binary knows about, including the
// ones currently hidden from the tab bar. Hidden entries stay in
// the list so their IDs and lookup paths still resolve — they just
// don't render or accept function-key activation.
func AllTabs() []Tab {
	return []Tab{
		{TabSettings, "F1", "Settings", false},
		{TabF2, "F2", "App Store", false},
		{TabF3, "F3", "Omarchy", false},
		{TabF4, "F4", "Dark", false},
		{TabF5, "F5", "Scripting", false},
		{TabF6, "F6", "—", true},
		{TabF7, "F7", "—", true},
		{TabF8, "F8", "—", true},
		{TabF9, "F9", "—", true},
		{TabF10, "F10", "—", true},
		{TabF11, "F11", "—", true},
		{TabF12, "F12", "—", true},
	}
}

// VisibleTabs returns the subset of AllTabs that should appear in
// the tab bar and accept function-key activation.
func VisibleTabs() []Tab {
	all := AllTabs()
	out := make([]Tab, 0, len(all))
	for _, t := range all {
		if !t.Hidden {
			out = append(out, t)
		}
	}
	return out
}

// TabFromKey resolves a function-key label (e.g. "F2") to its tab
// ID. Hidden tabs are intentionally excluded so their keys become
// no-ops in the UI.
func TabFromKey(key string) (TabID, bool) {
	for _, t := range AllTabs() {
		if t.Hidden {
			continue
		}
		if strings.EqualFold(t.Key, key) {
			return t.ID, true
		}
	}
	return TabSettings, false
}

func ParseStartTab(args []string) TabID {
	const flag = "--tab="
	for _, a := range args {
		if strings.HasPrefix(a, flag) {
			if id, ok := TabFromKey(strings.TrimPrefix(a, flag)); ok {
				return id
			}
		}
	}
	return TabSettings
}
