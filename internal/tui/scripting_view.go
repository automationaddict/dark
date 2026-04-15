package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/help"
)

// renderScriptingTab is the top-level F5 renderer. The outer sidebar
// lists user scripts and the three reference groups. When the user
// lands on MCP / Lua / API, the content pane grows its own inner
// sub-nav (like F1 wifi's Adapters/Networks/Known column) and the
// doc area on the right renders markdown for the selected command.
func renderScriptingTab(s *core.State, width, height int) string {
	sidebar := renderScriptingOuterSidebar(s, height)
	contentWidth := width - lipgloss.Width(sidebar)
	content := renderScriptingContent(s, contentWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// renderScriptingOuterSidebar renders the F5 outer sidebar: two
// groups ("SCRIPTS" and "REFERENCE") with headers between them.
// It is short enough to never scroll; the long command lists all
// live in the inner sub-nav.
func renderScriptingOuterSidebar(s *core.State, height int) string {
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	headerStyle := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Width(itemWidth).
		PaddingLeft(1)

	selected := s.ScriptingOuterIndex()
	outerFocused := !s.ContentFocused

	var rows []string
	flat := 0

	rows = append(rows, headerStyle.Render("SCRIPTS"))

	rows = append(rows, renderOuterRow("+ New script", flat == selected, outerFocused, true, item, active, dim))
	flat++

	if len(s.Scripts) == 0 {
		rows = append(rows, item.Render(dim.Render("(none)")))
	}
	for _, sc := range s.Scripts {
		rows = append(rows, renderOuterRow(sc.Name, flat == selected, outerFocused, false, item, active, dim))
		flat++
	}

	rows = append(rows, "")
	rows = append(rows, headerStyle.Render("REFERENCE"))

	for _, label := range []string{"MCP", "Lua", "API"} {
		rows = append(rows, renderOuterRow(label, flat == selected, outerFocused, false, item, active, dim))
		flat++
	}

	body := strings.Join(rows, "\n")
	return renderSidebarPane(height, body, outerFocused)
}

func renderOuterRow(label string, selected, outerFocused, dimmed bool, item, active lipgloss.Style, dim lipgloss.Style) string {
	if selected {
		if outerFocused {
			return active.Render(label)
		}
		// Selected but unfocused — fall back to the dim marker so
		// it still reads as "current" without looking active.
		return item.Render(dim.Render(label))
	}
	if dimmed {
		return item.Render(dim.Render(label))
	}
	return item.Render(label)
}

// renderScriptingContent dispatches the content pane. Script rows
// render a file detail doc; MCP / Lua / API rows render a two-column
// layout: inner sub-nav + markdown doc for the selected row.
func renderScriptingContent(s *core.State, width, height int) string {
	switch s.ScriptingSelection.Kind {
	case core.SelKindNewScript:
		return renderScriptingMarkdown(width, height, newScriptMarkdown())
	case core.SelKindScript:
		return renderScriptingScriptDetail(s, width, height)
	case core.SelKindMCP:
		return renderScriptingRefSection(s, width, height, refMCP)
	case core.SelKindLua:
		return renderScriptingRefSection(s, width, height, refLua)
	case core.SelKindAPI:
		return renderScriptingRefSection(s, width, height, refAPI)
	}
	return renderContentPane(width, height, placeholderStyle.Render("Nothing selected."))
}

type refKind int

const (
	refMCP refKind = iota
	refLua
	refAPI
)

// renderScriptingRefSection composes the inner sub-nav on the left
// of the content pane with the markdown doc area on the right. The
// inner sub-nav is scrollable so the 100+ API subjects remain
// navigable without overflowing the viewport.
func renderScriptingRefSection(s *core.State, width, height int, kind refKind) string {
	inner := renderScriptingInnerSidebar(s, height, kind)
	docWidth := width - lipgloss.Width(inner)
	doc := renderScriptingRefDoc(s, docWidth, height, kind)
	return lipgloss.JoinHorizontal(lipgloss.Top, inner, doc)
}

// renderScriptingRefDoc renders the markdown doc for whichever
// command is currently highlighted in the inner sub-nav.
func renderScriptingRefDoc(s *core.State, width, height int, kind refKind) string {
	var src string
	switch kind {
	case refMCP:
		src = mcpMarkdown()
	case refLua:
		src = luaEntryMarkdown(s)
	case refAPI:
		src = apiEntryMarkdown(s)
	}
	return renderScriptingMarkdown(width, height, src)
}

// renderScriptingMarkdown glamour-renders the given markdown source
// inside a standard content pane.
func renderScriptingMarkdown(width, height int, src string) string {
	inner := width - 6
	if inner < 20 {
		inner = 20
	}
	rendered, err := help.RenderMarkdown(src, inner)
	if err != nil {
		return renderContentPane(width, height,
			placeholderStyle.Render("markdown render failed: "+err.Error()))
	}
	return renderContentPane(width, height, strings.TrimRight(rendered, "\n"))
}

func renderScriptingScriptDetail(s *core.State, width, height int) string {
	if !s.ScriptsLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading scripts…"))
	}
	idx := s.ScriptingSelection.Index
	if idx < 0 || idx >= len(s.Scripts) {
		return renderContentPane(width, height,
			placeholderStyle.Render("Script not found."))
	}
	sc := s.Scripts[idx]

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", sc.Name)
	fmt.Fprintf(&b, "**Path:** `%s`  \n", sc.Path)
	fmt.Fprintf(&b, "**Size:** %d bytes  \n", sc.SizeBytes)
	fmt.Fprintf(&b, "**Modified:** %s\n\n", sc.ModTime.Format(time.RFC3339))

	preview := sc.Preview
	if preview == "" {
		preview = "(empty)"
	} else {
		lines := strings.Split(preview, "\n")
		if len(lines) > 40 {
			lines = append(lines[:40], "-- … (truncated)")
		}
		preview = strings.Join(lines, "\n")
	}
	fmt.Fprintf(&b, "```lua\n%s\n```\n\n", preview)
	fmt.Fprintf(&b, "Press **Enter** to open the editor. Press **d** to delete.\n")

	return renderScriptingMarkdown(width, height, b.String())
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
