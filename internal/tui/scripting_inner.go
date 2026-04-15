package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

// renderScriptingInnerSidebar renders the command list inside the
// content pane for MCP / Lua / API sections. Headers (domain groups
// in the API catalog) are visual-only and are skipped by navigation.
// The visible window centers on the current inner selection so long
// lists stay navigable without overflowing the viewport.
func renderScriptingInnerSidebar(s *core.State, height int, kind refKind) string {
	rows, selectedRow := buildInnerRows(s, kind)
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	subHeader := lipgloss.NewStyle().
		Foreground(colorDim).
		Width(itemWidth).
		PaddingLeft(1)

	viewH := height - 4
	if viewH < 5 {
		viewH = 5
	}
	start := selectedRow - viewH/2
	if start < 0 {
		start = 0
	}
	maxStart := len(rows) - viewH
	if maxStart < 0 {
		maxStart = 0
	}
	if start > maxStart {
		start = maxStart
	}
	end := start + viewH
	if end > len(rows) {
		end = len(rows)
	}

	innerFocused := s.ContentFocused
	var out []string
	for i := start; i < end; i++ {
		r := rows[i]
		if r.header {
			if r.label == "" {
				out = append(out, item.Render(""))
			} else {
				out = append(out, subHeader.Render(r.label))
			}
			continue
		}
		label := r.label
		switch {
		case i == selectedRow && innerFocused:
			out = append(out, active.Render(label))
		case i == selectedRow:
			out = append(out, item.Render(dim.Render(label)))
		default:
			out = append(out, item.Render(label))
		}
	}
	if len(out) > 0 && start > 0 {
		out[0] = subHeader.Render("↑")
	}
	if len(out) > 0 && end < len(rows) {
		out[len(out)-1] = subHeader.Render("↓")
	}
	return renderSidebarPane(height, strings.Join(out, "\n"), innerFocused)
}

type innerRow struct {
	label  string
	header bool
}

// splitActionName parses a Lua action name like
// "dark.actions.wifi.scan" into its domain and verb so the sub-nav
// can group actions by domain. Any name that doesn't match falls
// back to a single "other" bucket.
func splitActionName(name string) (domain, verb string) {
	trimmed := strings.TrimPrefix(name, "dark.actions.")
	idx := strings.IndexByte(trimmed, '.')
	if idx < 0 {
		return "other", trimmed
	}
	return trimmed[:idx], trimmed[idx+1:]
}

func buildInnerRows(s *core.State, kind refKind) ([]innerRow, int) {
	var rows []innerRow
	selectedRow := 0
	switch kind {
	case refMCP:
		rows = append(rows, innerRow{label: "(coming soon)"})
		selectedRow = 0
	case refLua:
		if len(s.LuaRegistry) == 0 {
			rows = append(rows, innerRow{label: "loading…", header: true})
			return rows, 0
		}
		target := clamp(s.LuaInnerIdx, 0, len(s.LuaRegistry)-1)
		var funcs, events, actions []int
		for i, e := range s.LuaRegistry {
			switch e.Kind {
			case "function":
				funcs = append(funcs, i)
			case "event":
				events = append(events, i)
			case "action":
				actions = append(actions, i)
			}
		}
		emit := func(i int, label string) {
			if i == target {
				selectedRow = len(rows)
			}
			rows = append(rows, innerRow{label: label})
		}
		if len(funcs) > 0 {
			rows = append(rows, innerRow{label: "FUNCTIONS", header: true})
			for _, i := range funcs {
				emit(i, s.LuaRegistry[i].Name)
			}
		}
		if len(events) > 0 {
			if len(rows) > 0 {
				rows = append(rows, innerRow{header: true})
			}
			rows = append(rows, innerRow{label: "EVENTS", header: true})
			for _, i := range events {
				emit(i, s.LuaRegistry[i].Name)
			}
		}
		if len(actions) > 0 {
			if len(rows) > 0 {
				rows = append(rows, innerRow{header: true})
			}
			rows = append(rows, innerRow{label: "ACTIONS", header: true})
			var lastDomain string
			for _, i := range actions {
				e := s.LuaRegistry[i]
				domain, verb := splitActionName(e.Name)
				if domain != lastDomain {
					rows = append(rows, innerRow{label: strings.ToUpper(domain), header: true})
					lastDomain = domain
				}
				emit(i, verb)
			}
		}
	case refAPI:
		if len(s.APICommands) == 0 {
			rows = append(rows, innerRow{label: "loading…", header: true})
			return rows, 0
		}
		target := clamp(s.APIInnerIdx, 0, len(s.APICommands)-1)
		var lastDomain string
		for i, c := range s.APICommands {
			if c.Domain != lastDomain {
				rows = append(rows, innerRow{label: strings.ToUpper(c.Domain), header: true})
				lastDomain = c.Domain
			}
			if i == target {
				selectedRow = len(rows)
			}
			rows = append(rows, innerRow{label: c.Verb})
		}
	}
	return rows, selectedRow
}
