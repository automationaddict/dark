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
type Editor struct {
	title  string
	area   textarea.Model
	submit EditorSubmitFunc
	closed bool
}

// EditorSubmitFunc is invoked when Ctrl+S is pressed. It receives the
// final text and can return a tea.Cmd to dispatch a bus request. A nil
// return value is fine — the editor just closes.
type EditorSubmitFunc func(content string) tea.Cmd

// NewEditor constructs an editor with the given title, pre-filled
// content, and submit callback. width and height are the full
// terminal dimensions; the editor uses them to size its textarea.
func NewEditor(title, content string, width, height int, submit EditorSubmitFunc) *Editor {
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
		title:  title,
		area:   ta,
		submit: submit,
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
// doesn't have to guess.
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

	body := strings.Join([]string{title, "", e.area.View(), "", hint}, "\n")
	return groupBoxSections("", []string{body}, width-2, colorAccent)
}
