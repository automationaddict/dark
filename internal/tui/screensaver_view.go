package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

// renderAppearanceScreensaverSection is the Appearance → Screensaver
// inner sub-section. It shows:
//
//   - An overall status banner (enabled / disabled, plus a
//     "previewing…" indicator while the preview child is running)
//   - A dependencies box so the user knows whether tte is installed
//     and whether their terminal is one omarchy-launch-screensaver
//     knows how to configure
//   - A content preview box rendering the first handful of lines of
//     the ASCII art file verbatim so they can see what the screensaver
//     will actually show
//   - A pointer row reminding the user the trigger timeout lives on
//     the Privacy page with the other idle timers
//   - A hint line advertising the action keys
func renderAppearanceScreensaverSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.ContentFocused
	borderColor := colorBorder
	if focused {
		borderColor = colorAccent
	}

	var blocks []string
	blocks = append(blocks, renderScreensaverStatus(s, innerWidth, borderColor))
	blocks = append(blocks, renderScreensaverDependencies(s, innerWidth))
	blocks = append(blocks, renderScreensaverContent(s, innerWidth))
	blocks = append(blocks, renderScreensaverHint())
	if s.ScreensaverActionError != "" {
		blocks = append(blocks, statusOfflineStyle.Render("  "+s.ScreensaverActionError))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderScreensaverStatus(s *core.State, total int, border lipgloss.Color) string {
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	stateText := boolIndicator(s.Screensaver.Enabled)
	if s.ScreensaverPreviewing {
		stateText = statusBusyStyle.Render("previewing…")
	}

	timeoutText := "—"
	if s.Privacy.ScreensaverTimeout > 0 {
		timeoutText = fmt.Sprintf("%ds (edit on Privacy → Screen Lock)", s.Privacy.ScreensaverTimeout)
	} else if s.PrivacyLoaded {
		timeoutText = "not configured (edit on Privacy → Screen Lock)"
	}

	lines := []string{
		label.Render("State") + value.Render(stateText),
		label.Render("Trigger") + value.Render(timeoutText),
	}
	return groupBoxSections("Status", lines, total, border)
}

func renderScreensaverDependencies(s *core.State, total int) string {
	label := detailLabelStyle.Width(14)
	value := detailValueStyle

	tteText := boolIndicator(s.Screensaver.TTEInstalled)
	if !s.Screensaver.TTEInstalled {
		tteText = lipgloss.NewStyle().Foreground(colorRed).Render("missing")
	}

	terminalText := s.Screensaver.TerminalName
	if terminalText == "" {
		terminalText = lipgloss.NewStyle().Foreground(colorRed).Render("not detected")
	} else if !s.Screensaver.Supported {
		terminalText = lipgloss.NewStyle().Foreground(colorGold).Render(terminalText+" (unsupported — needs alacritty, ghostty, or kitty)")
	} else {
		terminalText = lipgloss.NewStyle().Foreground(colorGreen).Render(terminalText)
	}

	lines := []string{
		label.Render("tte") + value.Render(tteText),
		label.Render("Terminal") + value.Render(terminalText),
	}
	return groupBoxSections("Dependencies", lines, total, colorBorder)
}

// contentPreviewLines caps the inline content preview so the box
// doesn't consume the whole screen for long ASCII art.
const contentPreviewLines = 8

func renderScreensaverContent(s *core.State, total int) string {
	content := s.Screensaver.Content
	if content == "" {
		body := placeholderStyle.Render("(no content — edit with c to create one)")
		return groupBoxSections("Content", []string{body}, total, colorBorder)
	}
	lines := strings.Split(content, "\n")
	shown := lines
	truncated := false
	if len(lines) > contentPreviewLines {
		shown = lines[:contentPreviewLines]
		truncated = true
	}

	preview := strings.Join(shown, "\n")
	if truncated {
		preview += "\n" + placeholderStyle.Render(fmt.Sprintf("  … +%d more lines", len(lines)-contentPreviewLines))
	}

	var pathLine string
	if s.Screensaver.ContentPath != "" {
		pathLine = placeholderStyle.Render("  " + s.Screensaver.ContentPath)
	}

	body := preview
	if pathLine != "" {
		body += "\n\n" + pathLine
	}
	return groupBoxSections("Content", []string{body}, total, colorBorder)
}

func renderScreensaverHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var parts []string
	parts = append(parts, accent.Render("e")+" toggle")
	parts = append(parts, accent.Render("c")+" edit content")
	parts = append(parts, accent.Render("p")+" preview (any key exits)")
	return dim.Render("  " + strings.Join(parts, "  "))
}
