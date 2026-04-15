package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Editor is a full-screen multi-line text editor overlay. It wraps
// bubbles/textarea and owns its own key routing while open. Unlike
// Dialog (fixed 48-col modal), Editor sizes to the full terminal so
// wider content like ASCII art banners has room to breathe.
//
// The submit callback receives the full edited string once the user
// presses Ctrl+S. Esc cancels without submitting. There's no dirty
// check — if the user pressed Esc, they meant to cancel.
//
// When language is non-empty, View() renders the buffer with chroma
// syntax highlighting — every line except the one the cursor is
// currently on (that line stays plain so textarea's native cursor
// math keeps working without visual-column gymnastics). The user
// sees color flash back onto a line the moment they move off it,
// which matches how most terminal editors with live highlighting
// behave.
type Editor struct {
	title    string
	language string
	area     textarea.Model
	submit   EditorSubmitFunc
	closed   bool
}

// EditorSubmitFunc is invoked when Ctrl+S is pressed. It receives the
// final text and can return a tea.Cmd to dispatch a bus request. A nil
// return value is fine — the editor just closes.
type EditorSubmitFunc func(content string) tea.Cmd

// NewEditor constructs a plain-text editor with no syntax
// highlighting. Used for the screensaver ASCII art banner where
// highlighting would be meaningless.
func NewEditor(title, content string, width, height int, submit EditorSubmitFunc) *Editor {
	return newEditorWithLang(title, LangNone, content, width, height, submit)
}

// NewEditorWithLanguage constructs an editor with chroma syntax
// highlighting for the named language. Accepted languages:
// LangJSON, LangJSONC, LangCSS. Any other value silently falls
// back to plain text — the View override re-highlights each frame
// but returns the plain area view when the language isn't known.
func NewEditorWithLanguage(title, language, content string, width, height int, submit EditorSubmitFunc) *Editor {
	return newEditorWithLang(title, language, content, width, height, submit)
}

func newEditorWithLang(title, language, content string, width, height int, submit EditorSubmitFunc) *Editor {
	ta := textarea.New()
	ta.SetValue(content)
	ta.Focus()
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // unlimited

	// Reserve space for title row + hint row + box borders + a little
	// breathing room so the textarea doesn't touch the terminal edge.
	areaWidth := width - 6
	if areaWidth < 20 {
		areaWidth = 20
	}
	areaHeight := height - 6
	if areaHeight < 5 {
		areaHeight = 5
	}
	ta.SetWidth(areaWidth)
	ta.SetHeight(areaHeight)

	return &Editor{
		title:    title,
		language: language,
		area:     ta,
		submit:   submit,
	}
}

// Closed reports whether the editor has been dismissed.
func (e *Editor) Closed() bool {
	return e == nil || e.closed
}

// Update dispatches a single tea message. Key events outside the
// cancel/submit pair are forwarded to the underlying textarea so
// arrows, delete, home/end, word-jump, and typing all work.
func (e *Editor) Update(msg tea.Msg) tea.Cmd {
	if e == nil {
		return nil
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		// Cancel: throw the edit away. Esc is the conventional
		// "never mind" key everywhere else in dark.
		if km.Type == tea.KeyEsc {
			e.closed = true
			return nil
		}
		// Submit: Ctrl+S. Using Enter would conflict with textarea's
		// normal "insert newline" semantics, which is essential for
		// ASCII art editing.
		if km.Type == tea.KeyCtrlS {
			content := e.area.Value()
			e.closed = true
			if e.submit != nil {
				return e.submit(content)
			}
			return nil
		}
		// Ctrl+C is the global "quit dark" shortcut; let it through
		// so the editor doesn't trap the user.
		if km.Type == tea.KeyCtrlC {
			e.closed = true
			return tea.Quit
		}
	}

	var cmd tea.Cmd
	e.area, cmd = e.area.Update(msg)
	return cmd
}

// View renders the editor as a titled box filling most of the screen.
// The hint line advertises the submit / cancel keys so the user
// doesn't have to guess. When a language is set, the body is a
// custom-rendered viewport with chroma-highlighted lines and a
// manually-placed cursor block on the active row; otherwise it
// falls back to textarea's native view so plain-text editing keeps
// all of textarea's usual behavior.
func (e *Editor) View(width, height int) string {
	if e == nil {
		return ""
	}

	title := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Render(e.title)

	hint := lipgloss.NewStyle().
		Foreground(colorDim).
		Render("Ctrl+S submit · Esc cancel")

	var body string
	if e.language == LangNone {
		body = strings.Join([]string{title, "", e.area.View(), "", hint}, "\n")
	} else {
		body = strings.Join([]string{title, "", e.renderHighlighted(), "", hint}, "\n")
	}
	return groupBoxSections("", []string{body}, width-2, colorAccent)
}

// renderHighlighted builds the syntax-highlighted editor body from
// the textarea's current buffer, cursor position, and viewport
// dimensions. Every line except the cursor row is rendered with
// chroma ANSI colors; the cursor row stays plain so a native
// cursor block can be placed without visual-column arithmetic.
func (e *Editor) renderHighlighted() string {
	content := e.area.Value()
	lines := strings.Split(content, "\n")

	highlighted, ok := highlightLines(content, e.language)
	if !ok {
		// Chroma failed — degrade to plain textarea view so the
		// user can still edit. Language flag effectively ignored.
		return e.area.View()
	}

	cursorRow := e.area.Line()
	info := e.area.LineInfo()
	cursorCol := info.ColumnOffset

	areaWidth := e.area.Width()
	areaHeight := e.area.Height()

	// Simple viewport: keep cursor centered-ish when the buffer
	// overflows the visible area. Not as smart as textarea's own
	// viewport but close enough for config-file-sized inputs.
	start := cursorRow - areaHeight/2
	if start < 0 {
		start = 0
	}
	maxStart := len(lines) - areaHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if start > maxStart {
		start = maxStart
	}
	end := start + areaHeight
	if end > len(lines) {
		end = len(lines)
	}

	// Line-number column: width is digit count of the largest
	// visible line number, dimmed, right-padded.
	gutterWidth := lineNumberWidth(len(lines))
	gutterStyle := lipgloss.NewStyle().Foreground(colorDim)
	cursorGutterStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	var rendered []string
	for i := start; i < end; i++ {
		num := lineNumberCell(i+1, gutterWidth)
		var gutter string
		if i == cursorRow {
			gutter = cursorGutterStyle.Render(num + " │ ")
		} else {
			gutter = gutterStyle.Render(num + " │ ")
		}

		var line string
		if i == cursorRow {
			// Plain text with a cursor block at cursorCol. The
			// block uses colorAccent background so it's visible
			// against any terminal background.
			line = renderCursorLine(lines[i], cursorCol, areaWidth-gutterWidth-3)
		} else if i < len(highlighted) {
			line = truncateStyledLine(highlighted[i], areaWidth-gutterWidth-3)
		} else {
			// Defensive: content and highlighted slice diverge
			// (shouldn't happen because highlightLines aligns
			// them). Fall back to plain.
			line = lines[i]
		}
		rendered = append(rendered, gutter+line)
	}

	// Pad short output so the box body height is stable.
	for len(rendered) < areaHeight {
		rendered = append(rendered, "")
	}
	return strings.Join(rendered, "\n")
}

// lineNumberWidth returns the digit count of the largest 1-based
// line number, minimum 2 so single-digit files still look like a
// gutter and not a scrappy offset.
func lineNumberWidth(total int) int {
	if total < 10 {
		return 2
	}
	w := 0
	for total > 0 {
		w++
		total /= 10
	}
	return w
}

// lineNumberCell renders a right-aligned line number string of the
// given width.
func lineNumberCell(n, width int) string {
	s := ""
	for v := n; v > 0; v /= 10 {
		s = string(rune('0'+(v%10))) + s
	}
	if s == "" {
		s = "0"
	}
	for len(s) < width {
		s = " " + s
	}
	return s
}

// renderCursorLine draws the cursor row in plain text with a
// lipgloss-styled block at cursorCol. Truncation keeps the line
// within the visible area. Cursor overshoot past the line end is
// drawn as a trailing block so the user can see where typing
// would land.
func renderCursorLine(line string, cursorCol, maxWidth int) string {
	cursorStyle := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	runes := []rune(line)
	if cursorCol > len(runes) {
		cursorCol = len(runes)
	}

	var before, at, after string
	if cursorCol < len(runes) {
		before = string(runes[:cursorCol])
		at = cursorStyle.Render(string(runes[cursorCol]))
		after = string(runes[cursorCol+1:])
	} else {
		before = line
		at = cursorStyle.Render(" ")
		after = ""
	}
	out := before + at + after
	if maxWidth > 0 && lipgloss.Width(out) > maxWidth {
		// Truncate from the left when the cursor would otherwise
		// scroll off the right edge. The cursor stays in view at
		// the cost of losing some leading characters.
		runes := []rune(before)
		visible := maxWidth - lipgloss.Width(at) - lipgloss.Width(after)
		if visible < 0 {
			visible = 0
		}
		if len(runes) > visible {
			before = "…" + string(runes[len(runes)-visible+1:])
		}
		out = before + at + after
	}
	return out
}

// truncateStyledLine clips a highlighted (ANSI-styled) line to
// maxWidth visible columns. lipgloss.Width correctly ignores
// ANSI escapes, so this works on chroma output directly.
func truncateStyledLine(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return line
	}
	if lipgloss.Width(line) <= maxWidth {
		return line
	}
	// Walk the string one rune at a time, building a prefix until
	// we hit maxWidth visible columns, skipping ANSI escape runs.
	var b strings.Builder
	visible := 0
	i := 0
	runes := []rune(line)
	for i < len(runes) {
		if runes[i] == 0x1b {
			// Copy the whole escape sequence verbatim.
			b.WriteRune(runes[i])
			i++
			for i < len(runes) && !isCSITerminator(runes[i]) {
				b.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				b.WriteRune(runes[i])
				i++
			}
			continue
		}
		if visible >= maxWidth-1 {
			b.WriteRune('…')
			break
		}
		b.WriteRune(runes[i])
		visible++
		i++
	}
	// Close any lingering style so the ellipsis doesn't inherit
	// token color from a clipped-mid-token escape.
	b.WriteString("\x1b[0m")
	return b.String()
}

func isCSITerminator(r rune) bool {
	return r >= 0x40 && r <= 0x7e
}
