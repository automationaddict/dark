package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appearance"
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

	var blocks []string
	blocks = append(blocks, renderAppearanceTheme(s.Appearance, innerWidth))
	blocks = append(blocks, renderAppearanceColors(s.Appearance, innerWidth))

	if bg := renderAppearanceBackgrounds(s.Appearance, innerWidth); bg != "" {
		blocks = append(blocks, bg)
	}

	blocks = append(blocks, renderAppearanceThemeHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderAppearanceThemeHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	return dim.Render("  " + accent.Render("t") + " change theme")
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

func renderAppearanceTheme(a appearance.Snapshot, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	themeName := a.Theme
	if themeName == "" {
		themeName = placeholderStyle.Render("unknown")
	} else {
		themeName = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(themeName)
	}
	lines = append(lines, label.Render("Active Theme")+themeName)
	lines = append(lines, label.Render("Themes")+
		value.Render(fmt.Sprintf("%d installed", len(a.Themes))))

	if a.IconTheme != "" {
		lines = append(lines, label.Render("Icon Theme")+value.Render(a.IconTheme))
	}
	lines = append(lines, label.Render("Icon Themes")+
		value.Render(fmt.Sprintf("%d available", len(a.IconThemes))))

	if a.CursorTheme != "" {
		lines = append(lines, label.Render("Cursor Theme")+value.Render(a.CursorTheme))
	}
	if a.CursorSize > 0 {
		lines = append(lines, label.Render("Cursor Size")+
			value.Render(fmt.Sprintf("%d", a.CursorSize)))
	}
	if a.KeyboardRGB != "" {
		swatch := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#" + a.KeyboardRGB)).
			Render("██")
		lines = append(lines, label.Render("Keyboard RGB")+
			swatch+" "+value.Render("#"+a.KeyboardRGB))
	}

	if len(a.Fonts) > 0 {
		lines = append(lines, label.Render("Font Families")+
			value.Render(fmt.Sprintf("%d installed", len(a.Fonts))))
	}

	return groupBoxSections("Theme", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceColors(a appearance.Snapshot, total int) string {
	c := a.Colors
	if c.Background == "" && c.Foreground == "" {
		return groupBoxSections("Colors", []string{
			placeholderStyle.Render("no color palette loaded"),
		}, total, colorBorder)
	}

	lw := 18
	label := detailLabelStyle.Width(lw)

	colorLine := func(name, hex string) string {
		if hex == "" {
			return label.Render(name) + placeholderStyle.Render("—")
		}
		swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
		return label.Render(name) + swatch + " " + detailValueStyle.Render(hex)
	}

	var main []string
	main = append(main, colorLine("Background", c.Background))
	main = append(main, colorLine("Foreground", c.Foreground))
	main = append(main, colorLine("Accent", c.Accent))
	main = append(main, colorLine("Cursor", c.Cursor))
	main = append(main, colorLine("Selection FG", c.SelectionForeground))
	main = append(main, colorLine("Selection BG", c.SelectionBackground))

	var ansi []string
	palette := []struct {
		name string
		hex  string
	}{
		{"0 Black", c.Color0}, {"1 Red", c.Color1},
		{"2 Green", c.Color2}, {"3 Yellow", c.Color3},
		{"4 Blue", c.Color4}, {"5 Magenta", c.Color5},
		{"6 Cyan", c.Color6}, {"7 White", c.Color7},
		{"8 Bright Black", c.Color8}, {"9 Bright Red", c.Color9},
		{"10 Bright Green", c.Color10}, {"11 Bright Yellow", c.Color11},
		{"12 Bright Blue", c.Color12}, {"13 Bright Magenta", c.Color13},
		{"14 Bright Cyan", c.Color14}, {"15 Bright White", c.Color15},
	}

	for _, p := range palette {
		ansi = append(ansi, colorLine(p.name, p.hex))
	}

	return groupBoxSections("Colors", []string{
		strings.Join(main, "\n"),
		strings.Join(ansi, "\n"),
	}, total, colorBorder)
}

func renderAppearanceGeneral(a appearance.Snapshot, total int) string {
	g := a.General
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	lines = append(lines, label.Render("Gaps Inner")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d", g.GapsIn)))
	lines = append(lines, label.Render("Gaps Outer")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d", g.GapsOut)))
	lines = append(lines, label.Render("Border Size")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d", g.BorderSize)))

	if g.ActiveBorder != "" {
		lines = append(lines, label.Render("Active Border")+value.Render(g.ActiveBorder))
	}
	if g.InactiveBorder != "" {
		lines = append(lines, label.Render("Inactive Border")+value.Render(g.InactiveBorder))
	}

	lines = append(lines, label.Render("Layout")+value.Render(g.LayoutName))
	lines = append(lines, label.Render("Resize on Border")+
		boolIndicator(g.ResizeOnBorder))

	return groupBoxSections("General", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceDecoration(a appearance.Snapshot, total int) string {
	d := a.Decoration
	lw := 18
	label := detailLabelStyle.Width(lw)

	var lines []string

	lines = append(lines, label.Render("Rounding")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%dpx", d.Rounding)))
	lines = append(lines, label.Render("Dim Inactive")+boolIndicator(d.DimInactive))
	if d.DimInactive {
		lines = append(lines, label.Render("Dim Strength")+
			detailValueStyle.Render(fmt.Sprintf("%.2f", d.DimStrength)))
	}

	return groupBoxSections("Decoration", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceBlur(a appearance.Snapshot, total int) string {
	b := a.Blur
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	lines = append(lines, label.Render("Blur")+boolIndicator(b.Enabled))
	lines = append(lines, label.Render("Size")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d", b.Size)))
	lines = append(lines, label.Render("Passes")+
		lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("%d", b.Passes)))
	lines = append(lines, label.Render("Brightness")+value.Render(fmt.Sprintf("%.2f", b.Brightness)))
	lines = append(lines, label.Render("Contrast")+value.Render(fmt.Sprintf("%.2f", b.Contrast)))
	lines = append(lines, label.Render("Special WS")+boolIndicator(b.Special))

	return groupBoxSections("Blur", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceShadow(a appearance.Snapshot, total int) string {
	sh := a.Shadow
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	lines = append(lines, label.Render("Shadow")+boolIndicator(sh.Enabled))
	lines = append(lines, label.Render("Range")+value.Render(fmt.Sprintf("%d", sh.Range)))
	lines = append(lines, label.Render("Render Power")+value.Render(fmt.Sprintf("%d", sh.RenderPower)))
	if sh.Color != "" {
		lines = append(lines, label.Render("Color")+value.Render(sh.Color))
	}

	return groupBoxSections("Shadow", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceAnimations(a appearance.Snapshot, total int) string {
	an := a.Animations
	lw := 18
	label := detailLabelStyle.Width(lw)

	var lines []string
	lines = append(lines, label.Render("Animations")+boolIndicator(an.Enabled))

	if len(an.Rules) > 0 {
		var ruleLines []string
		nameW := 20
		for _, r := range an.Rules {
			on := lipgloss.NewStyle().Foreground(colorGreen).Render("●")
			if !r.On {
				on = lipgloss.NewStyle().Foreground(colorDim).Render("○")
			}
			name := lipgloss.NewStyle().Foreground(colorDim).Width(nameW).Render(r.Name)
			detail := r.Speed
			if r.Curve != "" {
				detail += " " + r.Curve
			}
			if r.Style != "" {
				detail += " " + r.Style
			}
			ruleLines = append(ruleLines, "  "+on+" "+name+detailValueStyle.Render(detail))
		}
		lines = append(lines, strings.Join(ruleLines, "\n"))
	}

	return groupBoxSections("Animations", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceLayout(a appearance.Snapshot, total int) string {
	l := a.Layout
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	lines = append(lines, label.Render("Pseudotile")+boolIndicator(l.Pseudotile))
	lines = append(lines, label.Render("Preserve Split")+boolIndicator(l.PreserveSplit))

	splitDir := "right"
	switch l.ForceSplit {
	case 0:
		splitDir = "follow mouse"
	case 1:
		splitDir = "left"
	}
	lines = append(lines, label.Render("Force Split")+value.Render(splitDir))

	return groupBoxSections("Layout (Dwindle)", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceCursor(a appearance.Snapshot, total int) string {
	c := a.Cursor
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	lines = append(lines, label.Render("Hide on Keypress")+boolIndicator(c.HideOnKeyPress))

	warpLabel := "disabled"
	switch c.WarpOnChangeWorkspace {
	case 1:
		warpLabel = "on workspace change"
	case 2:
		warpLabel = "always"
	}
	lines = append(lines, label.Render("Cursor Warp")+value.Render(warpLabel))

	return groupBoxSections("Cursor", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceGroupbar(a appearance.Snapshot, total int) string {
	gb := a.Groupbar
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	lines = append(lines, label.Render("Font")+value.Render(gb.FontFamily))
	lines = append(lines, label.Render("Font Size")+value.Render(fmt.Sprintf("%d", gb.FontSize)))
	lines = append(lines, label.Render("Height")+value.Render(fmt.Sprintf("%dpx", gb.Height)))
	lines = append(lines, label.Render("Gaps")+value.Render(fmt.Sprintf("%d", gb.GapsIn)))
	lines = append(lines, label.Render("Gradients")+boolIndicator(gb.Gradients))

	return groupBoxSections("Groupbar", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderAppearanceBackgrounds(a appearance.Snapshot, total int) string {
	if len(a.Backgrounds) == 0 {
		return ""
	}

	var lines []string
	for i, name := range a.Backgrounds {
		idx := lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("%d ", i))
		lines = append(lines, "  "+idx+detailValueStyle.Render(name))
	}

	return groupBoxSections("Backgrounds", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func boolIndicator(v bool) string {
	if v {
		return lipgloss.NewStyle().Foreground(colorGreen).Render("enabled")
	}
	return lipgloss.NewStyle().Foreground(colorDim).Render("disabled")
}
