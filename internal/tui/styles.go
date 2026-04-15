package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/theme"
)

// Color slots are populated at startup by ApplyPalette from the loaded Omarchy
// theme. They're package-level vars (not consts) so the render layer can
// reference them by name without having to pass the palette around.
var (
	colorBg     lipgloss.Color
	colorHelpBg lipgloss.Color
	colorPanel  lipgloss.Color
	colorBorder lipgloss.Color
	colorMuted  lipgloss.Color // same as Border — used in helpPanel border references
	colorDim    lipgloss.Color // readable dimmed text for labels, hints
	colorText   lipgloss.Color
	colorAccent lipgloss.Color
	colorGreen  lipgloss.Color
	colorGold   lipgloss.Color
	colorRed    lipgloss.Color
)

// Styles the view layer renders with. Declared up front so existing render
// code can reference them as package-level names; the concrete values are
// filled in by ApplyPalette.
var (
	appStyle lipgloss.Style

	sidebarStyle      lipgloss.Style
	sidebarItem       lipgloss.Style
	sidebarItemActive lipgloss.Style

	contentStyle lipgloss.Style
	contentTitle lipgloss.Style

	tabBarStyle   lipgloss.Style
	tabItem       lipgloss.Style
	tabItemActive lipgloss.Style

	placeholderStyle lipgloss.Style

	statusBarStyle       lipgloss.Style
	statusErrorStyle     lipgloss.Style
	statusBusyStyle      lipgloss.Style
	statusOnlineStyle    lipgloss.Style
	statusOfflineStyle   lipgloss.Style

	cardStyle       lipgloss.Style
	cardTitleStyle  lipgloss.Style
	fieldLabelStyle lipgloss.Style
	fieldValueStyle lipgloss.Style

	groupBoxBorderStyle lipgloss.Style
	tableHeaderStyle    lipgloss.Style
	tableCellStyle          lipgloss.Style
	tableCellAccent         lipgloss.Style
	tableCellSelected       lipgloss.Style
	tableSelectionMarker    lipgloss.Style
	tableSelectionMarkerDim lipgloss.Style
	detailLabelStyle    lipgloss.Style
	detailValueStyle    lipgloss.Style

	helpPanelStyle         lipgloss.Style
	helpTitleStyle         lipgloss.Style
	helpDividerStyle       lipgloss.Style
	helpTocItemStyle       lipgloss.Style
	helpTocItemActiveStyle lipgloss.Style
	helpTocMutedStyle      lipgloss.Style
	helpFooterStyle        lipgloss.Style
	helpSearchStyle        lipgloss.Style
	helpMatchStyle         lipgloss.Style
	helpPanelErrorStyle    lipgloss.Style

	audioBarFilledStyle   lipgloss.Style
	audioBarMutedStyle    lipgloss.Style
	audioBarEmptyStyle    lipgloss.Style
	audioMeterFilledStyle lipgloss.Style
	audioMeterWarmStyle   lipgloss.Style
	audioMeterHotStyle    lipgloss.Style
	audioMeterDimStyle    lipgloss.Style

	dialogTitleStyle            lipgloss.Style
	dialogFieldLabelStyle       lipgloss.Style
	dialogFieldLabelActiveStyle lipgloss.Style
	dialogFieldStyle            lipgloss.Style
	dialogFieldActiveStyle      lipgloss.Style
	dialogHintStyle             lipgloss.Style
)

// ApplyPalette populates all colors and styles from the given palette. Must
// be called before any view renders.
func ApplyPalette(p theme.Palette) {
	colorBg = lipgloss.Color(p.Background)
	colorHelpBg = lipgloss.Color(p.HelpBackground)
	colorPanel = lipgloss.Color(p.Background)
	colorBorder = lipgloss.Color(p.Muted)
	colorMuted = lipgloss.Color(p.Muted)
	colorDim = lipgloss.Color(p.Dim)
	colorText = lipgloss.Color(p.Foreground)
	colorAccent = lipgloss.Color(p.Accent)
	colorGreen = lipgloss.Color(p.Green)
	colorGold = lipgloss.Color(p.Gold)
	colorRed = lipgloss.Color(p.Red)

	// Rebuild the chroma syntax-highlighting style so the config
	// editors (Top Bar config.jsonc / style.css) pick up the same
	// Omarchy palette the rest of the TUI uses.
	setHighlightPalette(p)

	appStyle = lipgloss.NewStyle().
		Background(colorBg).
		Foreground(colorText)

	sidebarStyle = lipgloss.NewStyle().
		Padding(1, 2).
		Background(colorHelpBg).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(colorBorder).
		BorderBackground(colorHelpBg)

	sidebarItem = lipgloss.NewStyle().
		Padding(0, 1).
		MarginBottom(1).
		Foreground(colorText).
		Background(colorHelpBg)

	sidebarItemActive = sidebarItem.
		Foreground(colorAccent).
		Background(colorHelpBg).
		Bold(true)

	contentStyle = lipgloss.NewStyle().
		Padding(1, 3).
		Foreground(colorText)

	contentTitle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		MarginBottom(1)

	tabBarStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(colorBorder)

	tabItem = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(colorDim)

	tabItemActive = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(colorAccent).
		Background(colorBg).
		Bold(true)

	placeholderStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Italic(true)

	statusBarStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(colorDim)

	statusErrorStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(colorRed).
		Bold(true)

	statusBusyStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(colorAccent).
		Bold(true)

	statusOnlineStyle = lipgloss.NewStyle().
		Foreground(colorGreen).
		Bold(true)

	statusOfflineStyle = lipgloss.NewStyle().
		Foreground(colorRed).
		Bold(true)

	cardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 2).
		MarginBottom(1)

	cardTitleStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		MarginBottom(1)

	fieldLabelStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Width(12)

	fieldValueStyle = lipgloss.NewStyle().
		Foreground(colorText)

	groupBoxBorderStyle = lipgloss.NewStyle().
		Foreground(colorBorder)

	tableHeaderStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Bold(false)

	tableCellStyle = lipgloss.NewStyle().
		Foreground(colorText)

	tableCellAccent = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	tableCellSelected = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	tableSelectionMarker = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	tableSelectionMarkerDim = lipgloss.NewStyle().
		Foreground(colorDim)

	detailLabelStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	detailValueStyle = lipgloss.NewStyle().
		Foreground(colorText)

	helpPanelStyle = lipgloss.NewStyle().
		Padding(0, 2).
		Background(colorBg).
		Foreground(colorText).
		Border(lipgloss.RoundedBorder(), true, false, true, true).
		BorderForeground(colorAccent).
		BorderBackground(colorBg)

	helpTitleStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBg).
		Bold(true).
		Padding(0, 0, 1, 0)

	helpDividerStyle = lipgloss.NewStyle().
		Foreground(colorBorder).
		Background(colorBg)

	helpTocItemStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Background(colorBg)

	helpTocItemActiveStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBg).
		Bold(true)

	helpTocMutedStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Background(colorBg).
		Italic(true)

	helpFooterStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Background(colorBg).
		Padding(1, 0, 0, 0)

	helpSearchStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBg).
		Padding(1, 0, 0, 0).
		Bold(true)

	helpMatchStyle = lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorGold).
		Bold(true)

	helpPanelErrorStyle = lipgloss.NewStyle().
		Foreground(colorRed).
		Background(colorBg).
		Italic(true)

	audioBarFilledStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	audioBarMutedStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	audioBarEmptyStyle = lipgloss.NewStyle().
		Foreground(colorBorder)

	audioMeterFilledStyle = lipgloss.NewStyle().
		Foreground(colorGreen)

	audioMeterWarmStyle = lipgloss.NewStyle().
		Foreground(colorGold)

	audioMeterHotStyle = lipgloss.NewStyle().
		Foreground(colorRed)

	audioMeterDimStyle = lipgloss.NewStyle().
		Foreground(colorBorder)

	dialogTitleStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	dialogFieldLabelStyle = lipgloss.NewStyle().
		Foreground(colorDim)

	dialogFieldLabelActiveStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	dialogFieldStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(colorBorder)

	dialogFieldActiveStyle = lipgloss.NewStyle().
		Foreground(colorText).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(colorAccent)

	dialogHintStyle = lipgloss.NewStyle().
		Foreground(colorDim).
		Italic(true)
}

// renderContentPane renders body inside contentStyle with exact dimensions.
// Height pads short content; MaxHeight truncates tall content.
func renderContentPane(width, height int, body string) string {
	return contentStyle.
		Width(width).MaxWidth(width).
		Height(height).MaxHeight(height).
		Render(body)
}

// renderScrollableContentPane renders body with vertical scrolling support.
// It slices the body lines by the scroll offset and updates state.ContentScrollMax.
func renderScrollableContentPane(s *core.State, width, height int, body string) string {
	lines := strings.Split(body, "\n")
	// contentStyle has Padding(1, 3) = 2 vertical padding rows
	viewH := height - 2
	if viewH < 1 {
		viewH = 1
	}
	maxScroll := len(lines) - viewH
	if maxScroll < 0 {
		maxScroll = 0
	}
	s.ContentScrollMax = maxScroll
	if s.ContentScroll > maxScroll {
		s.ContentScroll = maxScroll
	}

	start := s.ContentScroll
	end := start + viewH
	if end > len(lines) {
		end = len(lines)
	}
	visible := strings.Join(lines[start:end], "\n")

	if maxScroll > 0 {
		scrollHint := lipgloss.NewStyle().Foreground(colorDim).Render(
			fmt.Sprintf(" ↕ %d/%d", s.ContentScroll+1, maxScroll+1))
		visible = visible + "\n" + scrollHint
	}

	return contentStyle.
		Width(width).MaxWidth(width).
		Height(height).MaxHeight(height).
		Render(visible)
}

// renderSidebarPane renders body inside sidebarStyle with exact height.
// When focused is true the right border highlights with the accent color.
func renderSidebarPane(height int, body string, focused ...bool) string {
	style := sidebarStyle
	if len(focused) > 0 && focused[0] {
		style = style.BorderForeground(colorAccent)
	}
	return style.
		Height(height).MaxHeight(height).
		Render(body)
}
