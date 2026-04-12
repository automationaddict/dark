package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appstore"
)

// renderAppstoreDetailPane fills the main pane with the full readout
// of one package. The block is a stack of header (name, version,
// badges), metadata rows (size, updated, maintainer, etc.), and list
// sections (depends, optional deps). Keeping the structure flat makes
// the file easy to reason about and matches the detail panels used
// elsewhere in the TUI.
func renderAppstoreDetailPane(s *core.State, width, height int) string {
	border := colorAccent
	if !s.AppstoreDetailLoaded {
		body := placeholderStyle.Render("Loading package details…")
		return groupBoxSections("Detail", []string{body}, width, border)
	}

	d := s.AppstoreDetail
	lines := make([]string, 0, 32)

	lines = append(lines, renderAppstoreDetailHeader(d, width-4))
	lines = append(lines, "")
	lines = append(lines, renderAppstoreDetailFields(d)...)

	if d.LongDesc != "" && d.LongDesc != d.Description {
		lines = append(lines, "")
		lines = append(lines, detailLabelStyle.Render("Description"))
		lines = append(lines, wrapText(d.LongDesc, width-6)...)
	}

	if len(d.Depends) > 0 {
		lines = append(lines, "")
		lines = append(lines, detailLabelStyle.Render("Depends On"))
		lines = append(lines, wrapList(d.Depends, width-6))
	}
	if len(d.OptDepends) > 0 {
		lines = append(lines, "")
		lines = append(lines, detailLabelStyle.Render("Optional"))
		lines = append(lines, wrapList(d.OptDepends, width-6))
	}
	if len(d.MakeDepends) > 0 {
		lines = append(lines, "")
		lines = append(lines, detailLabelStyle.Render("Make Deps"))
		lines = append(lines, wrapList(d.MakeDepends, width-6))
	}
	if len(d.Conflicts) > 0 {
		lines = append(lines, "")
		lines = append(lines, detailLabelStyle.Render("Conflicts"))
		lines = append(lines, wrapList(d.Conflicts, width-6))
	}

	// Pad to full height so the box doesn't shrink when the package
	// has sparse metadata.
	for len(lines) < height-2 {
		lines = append(lines, "")
	}
	// And truncate if we overflow — the user can close and scroll
	// results, and phase 2 can add a scroll offset if this turns out
	// to be a problem in practice.
	if len(lines) > height-2 {
		lines = lines[:height-2]
		lines[len(lines)-1] = placeholderStyle.Render("… (detail truncated — resize terminal or press esc)")
	}

	body := strings.Join(lines, "\n")
	return groupBoxSections("Detail — "+d.Name, []string{body}, width, border)
}

// renderAppstoreDetailHeader is the single eye-catching row at the
// top of the detail pane: the package name in accent, version in
// dim, and an origin + install badge on the right.
func renderAppstoreDetailHeader(d appstore.Detail, width int) string {
	name := tableCellAccent.Render(d.Name)
	version := lipgloss.NewStyle().Foreground(colorDim).Render(d.Version)
	badge := appstoreOriginBadge(d.Package, 10)

	left := name + "  " + version
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(badge)
	padW := width - leftW - rightW
	if padW < 1 {
		padW = 1
	}
	return left + strings.Repeat(" ", padW) + badge
}

// renderAppstoreDetailFields returns the two-column "label : value"
// rows under the header. Missing fields render as "—" so the column
// alignment stays stable across packages.
func renderAppstoreDetailFields(d appstore.Detail) []string {
	kv := [][2]string{
		{"Repo", nonEmpty(d.Repo)},
		{"URL", nonEmpty(d.URL)},
		{"Licenses", nonEmpty(strings.Join(d.Licenses, ", "))},
		{"Download", appstore.HumanSize(d.DownloadSize)},
		{"Installed", appstore.HumanSize(d.InstalledSize)},
		{"Updated", appstore.RelativeTime(d.LastUpdatedUnix)},
	}
	if d.Origin == appstore.OriginAUR {
		kv = append(kv,
			[2]string{"Votes", fmt.Sprintf("%d", d.Votes)},
			[2]string{"Popularity", fmt.Sprintf("%.2f", d.Popularity)},
			[2]string{"Maintainer", nonEmpty(d.Maintainer)},
		)
	} else {
		if d.Packager != "" {
			kv = append(kv, [2]string{"Packager", d.Packager})
		}
	}

	rows := make([]string, 0, len(kv))
	labelW := 11
	for _, row := range kv {
		label := detailLabelStyle.Width(labelW).Render(row[0])
		value := detailValueStyle.Render(row[1])
		rows = append(rows, label+" "+value)
	}
	return rows
}

// nonEmpty returns s unchanged if it has content, otherwise the dash
// sentinel used across the TUI for "unknown / not reported".
func nonEmpty(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// wrapText splits a paragraph into wrapped lines no wider than max.
// Uses a naive word wrap which is fine for the descriptions shown here
// — none of them are long enough to warrant a full reflow library.
func wrapText(s string, max int) []string {
	if max < 20 {
		max = 20
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var line string
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		if lipgloss.Width(line)+1+lipgloss.Width(w) > max {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// wrapList folds a list of short tokens (dep names, license ids,
// etc.) into a single comma-separated string that soft-wraps on word
// boundaries. Returned as one string with embedded newlines so the
// caller can append it directly to the body slice.
func wrapList(items []string, max int) string {
	joined := strings.Join(items, ", ")
	lines := wrapText(joined, max)
	return strings.Join(lines, "\n")
}
