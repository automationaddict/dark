package help

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/charmbracelet/glamour"

	"github.com/johnnelson/dark/internal/theme"
)

//go:embed content/*.md
var fs embed.FS

//go:embed content/theme.json.tmpl
var themeTemplateSrc string

// Runtime state populated by SetPalette. The view layer has to call
// SetPalette before the first Load() or the help panel will render with
// pure fallback colors.
var (
	palette     theme.Palette
	paletteOnce sync.Once
	themeBytes  []byte
	bodyBg      string
	panelBg     string
)

// SetPalette tells the help package which colors to use for glamour rendering
// and ANSI background re-assertion. It's safe to call multiple times: later
// calls re-derive the rendered theme and backgrounds.
func SetPalette(p theme.Palette) {
	palette = p
	themeBytes = renderThemeBytes(p)
	bodyBg = theme.BgEscape(p.HelpBackground)
	panelBg = theme.BgEscape(p.Background)
}

func ensurePalette() {
	paletteOnce.Do(func() {
		if themeBytes == nil {
			SetPalette(theme.Load())
		}
	})
}

// PanelBgEscape is the ANSI background for the help panel chrome, sourced
// from the Omarchy palette's main background.
func PanelBgEscape() string {
	ensurePalette()
	return panelBg
}

// BodyBgEscape is the ANSI background for the scrollable body area, which is
// the palette background darkened a notch so the text viewport reads as
// inset.
func BodyBgEscape() string {
	ensurePalette()
	return bodyBg
}

// ReapplyPanelBackground walks a composed help panel string and ensures every
// cell has the panel background attribute set, so no ANSI reset inside the
// rendered frame can leak the terminal default color through.
func ReapplyPanelBackground(s string) string {
	return reapplyBackground(s, PanelBgEscape())
}

type TocEntry struct {
	Level int
	Title string
	Line  int
}

type Document struct {
	Key      string
	Title    string
	Lines    []string
	TOC      []TocEntry
	rawLower []string
}

func Load(key string, width int) (*Document, error) {
	ensurePalette()

	if width <= 0 {
		width = 40
	}

	raw, err := readMarkdown(key)
	if err != nil {
		return nil, err
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(themeBytes),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return nil, err
	}
	rendered, err := renderer.Render(raw)
	if err != nil {
		return nil, err
	}

	rendered = reapplyBackground(rendered, bodyBg)

	lines := splitLines(rendered)
	toc := extractTOC(raw, lines)
	title := extractTitle(raw, key)

	lower := make([]string, len(lines))
	for i, l := range lines {
		lower[i] = strings.ToLower(stripANSI(l))
	}

	return &Document{
		Key:      key,
		Title:    title,
		Lines:    lines,
		TOC:      toc,
		rawLower: lower,
	}, nil
}

func renderThemeBytes(p theme.Palette) []byte {
	tmpl, err := template.New("theme").Parse(themeTemplateSrc)
	if err != nil {
		return nil
	}
	data := struct {
		Fg, Accent, Dim, Muted, Border, Gold string
	}{
		Fg:     p.Foreground,
		Accent: p.Accent,
		Dim:    p.Dim,
		Muted:  p.Muted,
		Border: p.Muted,
		Gold:   p.Gold,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil
	}
	return buf.Bytes()
}

func readMarkdown(key string) (string, error) {
	b, err := fs.ReadFile("content/" + key + ".md")
	if err != nil {
		b, err = fs.ReadFile("content/default.md")
		if err != nil {
			return "", fmt.Errorf("no help content for %q", key)
		}
	}
	return string(b), nil
}

func extractTitle(raw, fallback string) string {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
	}
	return fallback
}

func extractTOC(raw string, rendered []string) []TocEntry {
	var entries []TocEntry

	stripped := make([]string, len(rendered))
	for i, l := range rendered {
		stripped[i] = stripANSI(l)
	}

	searchFrom := 0
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		level := 0
		for level < len(line) && line[level] == '#' {
			level++
		}
		if level == 0 || level > 3 {
			continue
		}
		title := strings.TrimSpace(line[level:])
		if title == "" {
			continue
		}
		lineIdx := findLine(stripped, title, searchFrom)
		if lineIdx < 0 {
			lineIdx = searchFrom
		}
		entries = append(entries, TocEntry{
			Level: level,
			Title: title,
			Line:  lineIdx,
		})
		searchFrom = lineIdx + 1
	}
	return entries
}

func findLine(rendered []string, needle string, from int) int {
	needle = strings.ToLower(strings.TrimSpace(needle))
	for i := from; i < len(rendered); i++ {
		hay := strings.ToLower(strings.TrimSpace(rendered[i]))
		if strings.Contains(hay, needle) {
			return i
		}
	}
	return -1
}

func (d *Document) Search(query string) []int {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var matches []int
	for i, l := range d.rawLower {
		if strings.Contains(l, q) {
			matches = append(matches, i)
		}
	}
	return matches
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	return strings.Split(s, "\n")
}

func reapplyBackground(s, bg string) string {
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bg)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = bg + line + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
