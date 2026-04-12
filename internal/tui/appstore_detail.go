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
		padLines := height - 3
		if padLines > 0 {
			body += strings.Repeat("\n", padLines)
		}
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

	viewH := height - 2
	if viewH < 1 {
		viewH = 1
	}

	// Store total lines so the state can clamp its scroll offset.
	s.AppstoreDetailLines = len(lines)
	s.AppstoreDetailViewH = viewH

	// Apply scroll window.
	scroll := s.AppstoreDetailScroll
	if scroll > len(lines)-viewH {
		scroll = len(lines) - viewH
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + viewH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[scroll:end]

	// Pad to full viewport height so the box doesn't shrink.
	for len(visible) < viewH {
		visible = append(visible, "")
	}

	// Scroll indicator in the title.
	title := "Detail — " + d.Name
	if len(lines) > viewH {
		title = fmt.Sprintf("Detail — %s  [%d/%d]", d.Name, scroll+1, len(lines)-viewH+1)
	}

	body := strings.Join(visible, "\n")
	return groupBoxSections(title, []string{body}, width, border)
}

// renderAppstoreDetailHeader is the single eye-catching row at the
// top of the detail pane: the package name in accent, version in
// dim, and an origin + install badge on the right.
func renderAppstoreDetailHeader(d appstore.Detail, width int) string {
	name := tableCellAccent.Render(d.Name)
	version := lipgloss.NewStyle().Foreground(colorDim).Render(d.Version)
	badge := appstoreOriginBadge(d.Package, 10)

	var action string
	if d.Installed {
		action = lipgloss.NewStyle().Foreground(colorRed).Render("  [X remove]")
	} else {
		action = lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("  [i install]")
	}

	left := name + "  " + version
	right := badge + action
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	padW := width - leftW - rightW
	if padW < 1 {
		padW = 1
	}
	return left + strings.Repeat(" ", padW) + right
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
