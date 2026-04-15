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

// renderScriptingOuterSidebar renders the F5 outer sidebar as four
// flat top-level items (Scripts / MCP / Lua / API). Each group's
// contents live in the content pane's inner sub-nav.
func renderScriptingOuterSidebar(s *core.State, height int) string {
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	selected := s.ScriptingOuterIndex()
	outerFocused := !s.ScriptingContentFocused

	labels := []string{"Scripts", "MCP", "Lua", "API"}
	var rows []string
	for i, label := range labels {
		rows = append(rows, renderOuterRow(label, i == selected, outerFocused, false, item, active, dim))
	}
	return renderSidebarPane(height, strings.Join(rows, "\n"), outerFocused)
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

// renderScriptingContent dispatches the content pane. Every group
// renders the same two-column layout: inner sub-nav on the left
// listing the group's entries, markdown doc area on the right.
func renderScriptingContent(s *core.State, width, height int) string {
	switch s.ScriptingSelection.Kind {
	case core.SelKindScripts:
		return renderScriptingRefSection(s, width, height, refScripts)
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
	refScripts refKind = iota
	refMCP
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
// entry is currently highlighted in the inner sub-nav.
func renderScriptingRefDoc(s *core.State, width, height int, kind refKind) string {
	switch kind {
	case refScripts:
		return renderScriptingScriptsDoc(s, width, height)
	case refMCP:
		return renderScriptingMarkdown(width, height, mcpMarkdown(s))
	case refLua:
		return renderScriptingMarkdown(width, height, luaEntryMarkdown(s))
	case refAPI:
		return renderScriptingMarkdown(width, height, apiEntryMarkdown(s))
	}
	return renderContentPane(width, height, placeholderStyle.Render("Nothing selected."))
}

// renderScriptingScriptsDoc is the Scripts group's doc area. Row 0
// is the `+ New script` affordance; rows 1..n render the file detail
// pane for the corresponding script.
func renderScriptingScriptsDoc(s *core.State, width, height int) string {
	if s.ScriptsInnerIdx == 0 {
		return renderScriptingMarkdown(width, height, newScriptMarkdown())
	}
	return renderScriptingScriptDetail(s, width, height)
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
	idx, ok := s.SelectedScriptIdx()
	if !ok {
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
	}
	fmt.Fprintf(&b, "```lua\n%s\n```\n", preview)

	inner := width - 6
	if inner < 20 {
		inner = 20
	}
	rendered, err := help.RenderMarkdown(b.String(), inner)
	if err != nil {
		return renderContentPane(width, height,
			placeholderStyle.Render("markdown render failed: "+err.Error()))
	}
	rendered = strings.TrimRight(rendered, "\n")

	// The script detail pane uses a dedicated style with zero bottom
	// padding so the footer "menu bar" sits flush with the bottom
	// edge of the content area. contentStyle's default Padding(1, 3)
	// would leave an unused row below the footer, so we swap in
	// Padding(1, 3, 0, 3) just for this render.
	paneStyle := lipgloss.NewStyle().
		Padding(1, 3, 0, 3).
		Foreground(colorText)

	footerStyle := lipgloss.NewStyle().Foreground(colorDim)
	footerText := "↑↓ select  ·  pgup/pgdn scroll  ·  enter edit  ·  d delete"

	// Inner body height = total - top pad (1) - divider (1) - footer (1).
	bodyH := height - 3
	if bodyH < 1 {
		bodyH = 1
	}

	lines := strings.Split(rendered, "\n")
	maxScroll := len(lines) - bodyH
	if maxScroll < 0 {
		maxScroll = 0
	}
	s.ScriptPreviewScrollMax = maxScroll
	if s.ScriptPreviewScroll > maxScroll {
		s.ScriptPreviewScroll = maxScroll
	}
	start := s.ScriptPreviewScroll
	end := start + bodyH
	if end > len(lines) {
		end = len(lines)
	}
	window := make([]string, 0, bodyH)
	window = append(window, lines[start:end]...)
	for len(window) < bodyH {
		window = append(window, "")
	}
	visible := strings.Join(window, "\n")

	divider := lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Repeat("─", inner))

	footer := footerStyle.Render(footerText)
	if maxScroll > 0 {
		indicator := footerStyle.Render(
			fmt.Sprintf("  (%d/%d)", s.ScriptPreviewScroll+1, maxScroll+1))
		footer += indicator
	}

	body := visible + "\n" + divider + "\n" + footer
	return paneStyle.
		Width(width).MaxWidth(width).
		Height(height).MaxHeight(height).
		Render(body)
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
