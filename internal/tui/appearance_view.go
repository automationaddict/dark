package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

func renderAppearance(s *core.State, width, height int) string {
	if !s.AppearanceLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading appearance info…"))
	}

	secs := core.AppearanceSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.AppearanceSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveAppearanceSection()
	var content string
	switch sec.ID {
	case "theme":
		content = renderAppearanceThemeSection(s, contentWidth, height)
	case "fonts":
		content = renderAppearanceFontsSection(s, contentWidth, height)
	case "windows":
		content = renderAppearanceWindowsSection(s, contentWidth, height)
	case "effects":
		content = renderAppearanceEffectsSection(s, contentWidth, height)
	case "cursor":
		content = renderAppearanceCursorSection(s, contentWidth, height)
	case "screensaver":
		content = renderAppearanceScreensaverSection(s, contentWidth, height)
	case "topbar":
		content = renderAppearanceTopBarSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Theme section ───────────────────────────────────────────────────

func renderAppearanceThemeSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	// Two-focus model: the Theme sub-section has a Theme box and
	// a Backgrounds box. When AppearanceContentFocused is off
	// (user still on the sub-section sidebar) both boxes show
	// plain borders. Once content focus is on, the currently-
	// focused box border accents so the user can tell which one
	// enter would act on.
	themeBorder := colorBorder
	bgBorder := colorBorder
	if s.AppearanceContentFocused {
		switch s.AppearanceThemeFocus {
		case core.AppearanceFocusTheme:
			themeBorder = colorAccent
		case core.AppearanceFocusBackgrounds:
			bgBorder = colorAccent
		}
	}

	var blocks []string
	blocks = append(blocks, renderAppearanceThemeBox(s.Appearance, innerWidth, themeBorder))
	blocks = append(blocks, renderAppearanceColors(s.Appearance, innerWidth))

	if bg := renderAppearanceBackgroundsBox(s, innerWidth, bgBorder); bg != "" {
		blocks = append(blocks, bg)
	}

	blocks = append(blocks, renderAppearanceThemeHint(s))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// renderAppearanceThemeHint tailors the hint line to the current
// focus state so the user always sees the next available key.
func renderAppearanceThemeHint(s *core.State) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	if !s.AppearanceContentFocused {
		return dim.Render("  " +
			accent.Render("enter") + " focus  " +
			accent.Render("t") + " change theme")
	}
	if s.AppearanceThemeFocus == core.AppearanceFocusBackgrounds {
		return dim.Render("  " +
			accent.Render("j/k") + " select  " +
			accent.Render("enter") + " set background  " +
			accent.Render("tab") + " theme box  " +
			accent.Render("esc") + " back")
	}
	return dim.Render("  " +
		accent.Render("enter") + " change theme  " +
		accent.Render("tab") + " backgrounds  " +
		accent.Render("esc") + " back")
}

// ── Fonts section ──────────────────────────────────────────────────

func renderAppearanceFontsSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	a := s.Appearance
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	current := a.CurrentFont
	if current == "" {
		current = "—"
	}
	sizeStr := "—"
	if a.CurrentFontSize > 0 {
		sizeStr = fmt.Sprintf("%dpt", a.CurrentFontSize)
	}

	focused := s.ContentFocused
	borderColor := colorBorder
	if focused {
		borderColor = colorAccent
	}

	var lines []string
	lines = append(lines, label.Render("Family")+value.Render(current))
	lines = append(lines, label.Render("Size")+value.Render(sizeStr))
	lines = append(lines, label.Render("Available")+
		value.Render(fmt.Sprintf("%d monospace", len(a.Fonts))))

	fontBox := groupBoxSections("Font", []string{
		strings.Join(lines, "\n"),
	}, innerWidth, borderColor)

	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	hint := dim.Render("  " +
		accent.Render("f") + " change font  " +
		accent.Render("+/-") + " font size")

	body := lipgloss.JoinVertical(lipgloss.Left, fontBox, "", hint)
	return renderContentPane(width, height, body)
}

// ── Windows section ─────────────────────────────────────────────────

func renderAppearanceWindowsSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderAppearanceGeneral(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceDecoration(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceLayout(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceWindowsHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderAppearanceWindowsHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("i/I")+" gaps in")
	hints = append(hints, accent.Render("o/O")+" gaps out")
	hints = append(hints, accent.Render("b")+" border")
	hints = append(hints, accent.Render("r/R")+" rounding")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Effects section ─────────────────────────────────────────────────

func renderAppearanceEffectsSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderAppearanceBlur(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceShadow(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceAnimations(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceEffectsHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderAppearanceEffectsHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("B")+" blur toggle")
	hints = append(hints, accent.Render("z/Z")+" blur size")
	hints = append(hints, accent.Render("x/X")+" blur passes")
	hints = append(hints, accent.Render("A")+" animations toggle")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Cursor section ──────────────────────────────────────────────────

func renderAppearanceCursorSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderAppearanceCursor(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceGroupbar(s.Appearance, innerWidth))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// ── Shared rendering helpers ────────────────────────────────────────

func boolIndicator(v bool) string {
	if v {
		return lipgloss.NewStyle().Foreground(colorGreen).Render("enabled")
	}
	return lipgloss.NewStyle().Foreground(colorDim).Render("disabled")
}
