package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/keybind"
)

func renderKeybindings(s *core.State, width, height int) string {
	if !s.KeybindingsLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading keybindings…"))
	}

	bindings := s.Keybindings.Bindings
	if len(bindings) == 0 {
		return renderContentPane(width, height,
			placeholderStyle.Render("No keybindings found."))
	}

	innerWidth := width - 6
	if innerWidth < 60 {
		innerWidth = 60
	}

	title := contentTitle.Render("Keybindings")

	numW := 5
	modsW := 18
	keyW := 14
	sourceW := 10
	descW := innerWidth - numW - modsW - keyW - sourceW - 5
	if descW < 16 {
		descW = 16
	}

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	// Chrome lines: padding(1) + title(1) + blank(1) + table header(3) +
	// table bottom border(1) + blank(1) + hint(1) + blank(1) + summary(1) +
	// padding(1) = 12
	chromeLines := 12
	maxVisible := height - chromeLines
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Window the data so the selected row is always visible.
	startIdx := 0
	if len(bindings) > maxVisible {
		startIdx = s.KeybindIdx - maxVisible/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+maxVisible > len(bindings) {
			startIdx = len(bindings) - maxVisible
		}
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(bindings) {
		endIdx = len(bindings)
	}

	var data [][]string
	for i := startIdx; i < endIdx; i++ {
		b := bindings[i]
		source := string(b.Source)
		if b.Source == keybind.SourceDefault {
			source = "default"
		}
		mods := b.Mods
		if mods == "" {
			mods = "—"
		}
		desc := b.Desc
		if desc == "" {
			desc = b.Dispatcher
		}
		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			mods,
			b.Key,
			desc,
			source,
		})
	}

	// Adjust the selected index relative to the visible window.
	visibleIdx := s.KeybindIdx - startIdx

	table := renderTable(
		[]string{"#", "Modifiers", "Key", "Description", "Source"},
		[]int{numW, modsW, keyW, descW, sourceW},
		data,
		visibleIdx, s.ContentFocused, selectedCell,
	)

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"enter edit · a add · d delete")

	// Summary with scroll position.
	userCount := 0
	for _, b := range bindings {
		if b.Source == keybind.SourceUser {
			userCount++
		}
	}
	summaryText := fmt.Sprintf("%d bindings (%d user, %d default)",
		len(bindings), userCount, len(bindings)-userCount)
	if len(bindings) > maxVisible {
		summaryText += fmt.Sprintf("  ↕ %d-%d/%d",
			startIdx+1, endIdx, len(bindings))
	}
	summary := lipgloss.NewStyle().Foreground(colorDim).Render(summaryText)

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "", table, "", hint, "", summary)

	return renderContentPane(width, height, strings.TrimRight(body, "\n"))
}
