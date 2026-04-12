package appstore

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HumanSize renders a byte count as a short human-readable string. Zero
// renders as "—" because the appstore UI treats zero as "unknown" — AUR
// packages that haven't been installed locally have no measurable size.
func HumanSize(bytes int64) string {
	if bytes <= 0 {
		return "—"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	value := float64(bytes) / float64(div)
	suffix := []string{"KiB", "MiB", "GiB", "TiB"}[exp]
	if value >= 100 {
		return fmt.Sprintf("%.0f %s", value, suffix)
	}
	if value >= 10 {
		return fmt.Sprintf("%.1f %s", value, suffix)
	}
	return fmt.Sprintf("%.2f %s", value, suffix)
}

// RelativeTime renders a Unix timestamp as "3 days ago" / "2 months ago"
// relative to now. Zero renders as "—". Future timestamps (which
// sometimes happen when pacman build dates drift on mirrored packages)
// fall through to an absolute date to avoid a nonsensical "-3 hours
// ago".
func RelativeTime(unix int64) string {
	if unix <= 0 {
		return "—"
	}
	t := time.Unix(unix, 0)
	diff := time.Since(t)
	if diff < 0 {
		return t.Format("2006-01-02")
	}
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		m := int(diff.Minutes())
		return fmt.Sprintf("%d minute%s ago", m, plural(m))
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		return fmt.Sprintf("%d hour%s ago", h, plural(h))
	case diff < 14*24*time.Hour:
		d := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", d, plural(d))
	case diff < 60*24*time.Hour:
		w := int(diff.Hours() / 24 / 7)
		return fmt.Sprintf("%d week%s ago", w, plural(w))
	case diff < 365*24*time.Hour:
		mo := int(diff.Hours() / 24 / 30)
		return fmt.Sprintf("%d month%s ago", mo, plural(mo))
	default:
		y := int(diff.Hours() / 24 / 365)
		return fmt.Sprintf("%d year%s ago", y, plural(y))
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ParseByteSize parses the size strings pacman -Si emits, e.g. "1934.33
// KiB" or "9.57 MiB". Returns zero on any parse error — the caller
// treats zero as unknown, which is the correct fallback semantics for
// this field.
func ParseByteSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil || value < 0 {
		return 0
	}
	var mult float64
	switch strings.ToLower(parts[1]) {
	case "b":
		mult = 1
	case "kib", "kb":
		mult = 1024
	case "mib", "mb":
		mult = 1024 * 1024
	case "gib", "gb":
		mult = 1024 * 1024 * 1024
	case "tib", "tb":
		mult = 1024 * 1024 * 1024 * 1024
	default:
		return 0
	}
	return int64(value * mult)
}

// SortResults orders search results for display. The order is:
//
//  1. Exact name match first (case-insensitive), so `firefox` ranks above
//     `firefox-developer-edition` for a "firefox" query.
//  2. Installed packages next, so users see what they already have.
//  3. Pacman repos before AUR, since the official repos are the trusted
//     path and AUR is opt-in.
//  4. Then by name, case-insensitive.
//
// The sort is stable so ties preserve the backend's insertion order.
func SortResults(pkgs []Package, query string) {
	q := strings.ToLower(strings.TrimSpace(query))
	sort.SliceStable(pkgs, func(i, j int) bool {
		ni := strings.ToLower(pkgs[i].Name)
		nj := strings.ToLower(pkgs[j].Name)
		if q != "" {
			exi := ni == q
			exj := nj == q
			if exi != exj {
				return exi
			}
		}
		if pkgs[i].Installed != pkgs[j].Installed {
			return pkgs[i].Installed
		}
		if pkgs[i].Origin != pkgs[j].Origin {
			return pkgs[i].Origin == OriginPacman
		}
		return ni < nj
	})
}

// TruncateDesc shortens a description to fit one row in the results
// list. The cut happens on the nearest word boundary and appends an
// ellipsis so the result always fits within max runes.
func TruncateDesc(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 1 || len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	cut := max - 1
	for cut > 0 && runes[cut] != ' ' {
		cut--
	}
	if cut == 0 {
		cut = max - 1
	}
	return strings.TrimRight(string(runes[:cut]), " .,") + "…"
}
