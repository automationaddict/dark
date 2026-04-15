package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/workspaces"
)

// renderWorkspaces is the top-level Workspaces settings view.
// Layout mirrors every other sub-section-bearing panel in dark:
// an inner sidebar on the left with the three sub-sections, the
// content pane on the right showing the active sub-section.
func renderWorkspaces(s *core.State, width, height int) string {
	if !s.WorkspacesLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading workspaces…"))
	}

	secs := core.WorkspacesSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.WorkspacesContentFocused
	_ = sidebarFocused
	sidebar := renderInnerSidebar(s, entries, s.WorkspacesSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	var content string
	switch s.ActiveWorkspacesSection().ID {
	case "overview":
		content = renderWorkspacesOverviewSection(s, contentWidth, height)
	case "layout":
		content = renderWorkspacesLayoutSection(s, contentWidth, height)
	case "behavior":
		content = renderWorkspacesBehaviorSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Overview sub-section ────────────────────────────────────────────

func renderWorkspacesOverviewSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	// Boxes only highlight once the user has stepped into the
	// content region (second enter). While the sub-section sidebar
	// is focused, ContentFocused is true but WorkspacesContentFocused
	// is false — we want the plain border in that intermediate
	// state so the user can visually tell they're still on the
	// sidebar.
	focused := s.WorkspacesContentFocused
	borderColor := colorBorder
	if focused {
		borderColor = colorAccent
	}

	title := contentTitle.Render("Workspaces")

	if len(s.Workspaces.Workspaces) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			placeholderStyle.Render("No workspaces reported by hyprctl."),
		)
		return renderContentPane(width, height, body)
	}

	table := renderWorkspacesTable(s, focused)

	summary := renderWorkspacesSummary(s, innerWidth)
	box := groupBoxSections("Live", []string{table}, innerWidth, borderColor)

	hint := renderWorkspacesOverviewHint()

	var blocks []string
	blocks = append(blocks, title, "", box, summary, "", hint)
	if s.WorkspacesActionError != "" {
		blocks = append(blocks, statusOfflineStyle.Render("  "+s.WorkspacesActionError))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// renderWorkspacesTable uses the shared accent-table renderer so
// it matches the wifi / bluetooth / network tables visually.
func renderWorkspacesTable(s *core.State, focused bool) string {
	cols := []accentColumn[workspaces.Workspace]{
		{"ID", func(w workspaces.Workspace) string { return fmt.Sprintf("%d", w.ID) },
			func(w workspaces.Workspace) bool { return w.ID == s.Workspaces.ActiveID }},
		{"Name", func(w workspaces.Workspace) string { return orDash(w.Name) }, nil},
		{"Monitor", func(w workspaces.Workspace) string { return orDash(w.Monitor) }, nil},
		{"Windows", func(w workspaces.Workspace) string { return fmt.Sprintf("%d", w.Windows) }, nil},
		{"Layout", func(w workspaces.Workspace) string { return orDash(w.TiledLayout) }, nil},
		{"Last focused", func(w workspaces.Workspace) string {
			if w.LastWindowTitle == "" {
				return "—"
			}
			return truncateStr(w.LastWindowTitle, 40)
		}, nil},
	}
	marker := func(sel, _ bool, w workspaces.Workspace) string {
		active := w.ID == s.Workspaces.ActiveID
		switch {
		case sel && active:
			return tableSelectionMarker.Render("▸ ")
		case sel:
			return tableSelectionMarkerDim.Render("▸ ")
		case active:
			return lipgloss.NewStyle().Foreground(colorAccent).Render("● ")
		default:
			return "  "
		}
	}
	return renderAccentTable(cols, s.Workspaces.Workspaces, s.WorkspacesContentIdx, focused, nil, marker)
}

func renderWorkspacesSummary(s *core.State, total int) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	active := 0
	for _, w := range s.Workspaces.Workspaces {
		if w.ID == s.Workspaces.ActiveID {
			active = w.ID
			break
		}
	}
	text := fmt.Sprintf("%d workspaces · active %d", len(s.Workspaces.Workspaces), active)
	if rules := len(s.Workspaces.Rules); rules > 0 {
		text += fmt.Sprintf(" · %d persistent rules", rules)
	}
	return dim.Render("  " + text)
}

func renderWorkspacesOverviewHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var parts []string
	parts = append(parts, accent.Render("j/k")+" select")
	parts = append(parts, accent.Render("enter")+" switch")
	parts = append(parts, accent.Render("r")+" rename")
	parts = append(parts, accent.Render("m")+" move to monitor")
	parts = append(parts, accent.Render("L")+" cycle layout")
	return dim.Render("  " + strings.Join(parts, "  "))
}

// ── Layout sub-section ──────────────────────────────────────────────

func renderWorkspacesLayoutSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	// Every Layout box is editable via the hint-line keys, so
	// they all accent-highlight together once the user has
	// stepped into the content region.
	borderColor := colorBorder
	if s.WorkspacesContentFocused {
		borderColor = colorAccent
	}

	label := detailLabelStyle.Width(18)
	value := detailValueStyle

	defaultLines := []string{
		label.Render("Default layout") + value.Render(orDash(s.Workspaces.DefaultLayout)),
	}
	defaultBox := groupBoxSections("Default", defaultLines, innerWidth, borderColor)

	dwindleLines := []string{
		label.Render("Pseudotile") + value.Render(boolIndicator(s.Workspaces.Dwindle.Pseudotile)),
		label.Render("Preserve split") + value.Render(boolIndicator(s.Workspaces.Dwindle.PreserveSplit)),
		label.Render("Force split") + value.Render(forceSplitLabel(s.Workspaces.Dwindle.ForceSplit)),
		label.Render("Smart split") + value.Render(boolIndicator(s.Workspaces.Dwindle.SmartSplit)),
		label.Render("Smart resizing") + value.Render(boolIndicator(s.Workspaces.Dwindle.SmartResizing)),
	}
	dwindleBox := groupBoxSections("Dwindle", dwindleLines, innerWidth, borderColor)

	masterLines := []string{
		label.Render("New window status") + value.Render(orDash(s.Workspaces.Master.NewStatus)),
		label.Render("Orientation") + value.Render(orDash(s.Workspaces.Master.Orientation)),
	}
	masterBox := groupBoxSections("Master", masterLines, innerWidth, borderColor)

	hint := renderWorkspacesLayoutHint()

	var blocks []string
	blocks = append(blocks, defaultBox, dwindleBox, masterBox, "", hint)
	if s.WorkspacesActionError != "" {
		blocks = append(blocks, statusOfflineStyle.Render("  "+s.WorkspacesActionError))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderWorkspacesLayoutHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var parts []string
	parts = append(parts, accent.Render("L")+" default layout")
	parts = append(parts, accent.Render("t")+" pseudotile")
	parts = append(parts, accent.Render("p")+" preserve split")
	parts = append(parts, accent.Render("f")+" force split")
	parts = append(parts, accent.Render("M")+" master status")
	return dim.Render("  " + strings.Join(parts, "  "))
}

func forceSplitLabel(n int) string {
	switch n {
	case 0:
		return "auto"
	case 1:
		return "left/top"
	case 2:
		return "right/bottom"
	default:
		return fmt.Sprintf("%d", n)
	}
}

// ── Behavior sub-section ────────────────────────────────────────────

func renderWorkspacesBehaviorSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	borderColor := colorBorder
	if s.WorkspacesContentFocused {
		borderColor = colorAccent
	}

	label := detailLabelStyle.Width(24)
	value := detailValueStyle

	lines := []string{
		label.Render("Cursor warp on change") + value.Render(boolIndicator(s.Workspaces.CursorWarp)),
		label.Render("Animations enabled") + value.Render(boolIndicator(s.Workspaces.AnimationsEnabled)),
		label.Render("Hide special on change") + value.Render(boolIndicator(s.Workspaces.HideSpecialOnChange)),
	}
	box := groupBoxSections("Behavior", lines, innerWidth, borderColor)

	hint := renderWorkspacesBehaviorHint()

	var blocks []string
	blocks = append(blocks, box, "", hint)
	if s.WorkspacesActionError != "" {
		blocks = append(blocks, statusOfflineStyle.Render("  "+s.WorkspacesActionError))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderWorkspacesBehaviorHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var parts []string
	parts = append(parts, accent.Render("c")+" cursor warp")
	parts = append(parts, accent.Render("a")+" animations")
	parts = append(parts, accent.Render("h")+" hide special")
	return dim.Render("  " + strings.Join(parts, "  "))
}

