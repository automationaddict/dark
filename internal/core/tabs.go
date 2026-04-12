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
	ID    TabID
	Key   string
	Title string
}

func AllTabs() []Tab {
	return []Tab{
		{TabSettings, "F1", "Settings"},
		{TabF2, "F2", "App Store"},
		{TabF3, "F3", "—"},
		{TabF4, "F4", "—"},
		{TabF5, "F5", "—"},
		{TabF6, "F6", "—"},
		{TabF7, "F7", "—"},
		{TabF8, "F8", "—"},
		{TabF9, "F9", "—"},
		{TabF10, "F10", "—"},
		{TabF11, "F11", "—"},
		{TabF12, "F12", "—"},
	}
}

func TabFromKey(key string) (TabID, bool) {
	for _, t := range AllTabs() {
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
