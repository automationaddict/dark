package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// DialogFieldKind distinguishes visible text fields from masked password
// fields. Add more kinds here (selects, checkboxes, multiline) as we grow
// the dialog component.
type DialogFieldKind int

const (
	DialogFieldText DialogFieldKind = iota
	DialogFieldPassword
)

// DialogFieldSpec defines one form row. Declared up front when building
// the dialog so the Dialog struct can carry internal state per field.
type DialogFieldSpec struct {
	Key   string
	Label string
	Kind  DialogFieldKind
	Value string // optional pre-filled value
}

// DialogResult is passed to the submit callback. Keys are the spec keys.
type DialogResult map[string]string

// DialogSubmitFunc is invoked when the user presses enter. It should
// return a tea.Cmd that performs the actual work (usually a bus request).
// Returning nil is fine — the dialog just closes.
type DialogSubmitFunc func(DialogResult) tea.Cmd

// Dialog is a modal form overlay. It owns its own focus ring and key
// routing; while a dialog is open, Model.Update funnels every key event
// into Dialog.Update.
type Dialog struct {
	title  string
	fields []dialogField
	submit DialogSubmitFunc
	active int
	closed bool
}

type dialogField struct {
	key   string
	label string
	kind  DialogFieldKind
	value []rune
}

// NewDialog constructs a dialog with the given title, field definitions,
// and submit callback.
func NewDialog(title string, fields []DialogFieldSpec, submit DialogSubmitFunc) *Dialog {
	d := &Dialog{title: title, submit: submit}
	for _, f := range fields {
		d.fields = append(d.fields, dialogField{
			key:   f.Key,
			label: f.Label,
			kind:  f.Kind,
			value: []rune(f.Value),
		})
	}
	return d
}

// Closed reports whether the dialog has been dismissed (either via
// esc/cancel or via a submit). Model checks this after each Update to
// clear its dialog pointer.
func (d *Dialog) Closed() bool { return d == nil || d.closed }

// Update dispatches a single key event. Returns any tea.Cmd produced by
// a submit callback (nil for navigation/typing/cancel).
func (d *Dialog) Update(msg tea.KeyMsg) tea.Cmd {
	if d == nil {
		return nil
	}
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		d.closed = true
		return nil
	case tea.KeyEnter:
		values := DialogResult{}
		for _, f := range d.fields {
			values[f.key] = string(f.value)
		}
		d.closed = true
		if d.submit != nil {
			return d.submit(values)
		}
		return nil
	case tea.KeyTab, tea.KeyDown:
		d.advanceField(1)
	case tea.KeyShiftTab, tea.KeyUp:
		d.advanceField(-1)
	case tea.KeyBackspace:
		d.backspace()
	case tea.KeyRunes, tea.KeySpace:
		d.appendRunes(msg.Runes)
	}
	return nil
}

func (d *Dialog) advanceField(delta int) {
	if len(d.fields) == 0 {
		return
	}
	d.active = (d.active + delta + len(d.fields)) % len(d.fields)
}

func (d *Dialog) backspace() {
	if len(d.fields) == 0 {
		return
	}
	f := &d.fields[d.active]
	if len(f.value) == 0 {
		return
	}
	f.value = f.value[:len(f.value)-1]
}

func (d *Dialog) appendRunes(runes []rune) {
	if len(d.fields) == 0 || len(runes) == 0 {
		return
	}
	f := &d.fields[d.active]
	f.value = append(f.value, runes...)
}

// View renders the dialog as a bordered modal box. Caller is responsible
// for compositing it onto the base view via overlayCenter.
func (d *Dialog) View() string {
	const innerWidth = 48

	// Title is drawn by groupBoxSections in the top border, not in the body.
	var rows []string

	for i, f := range d.fields {
		label := dialogFieldLabelStyle.Render(f.label)
		if i == d.active {
			label = dialogFieldLabelActiveStyle.Render(f.label)
		}
		rows = append(rows, label)
		rows = append(rows, renderDialogField(f, i == d.active, innerWidth))
		rows = append(rows, "")
	}

	rows = append(rows, dialogHintStyle.Render("enter submit · tab next field · esc cancel"))

	body := strings.Join(rows, "\n")
	return groupBoxSections(d.title, []string{body}, innerWidth+4, colorAccent)
}

// renderDialogField prints one input row with the current value (or
// bullets for password fields) and a trailing cursor block when this
// field has focus.
func renderDialogField(f dialogField, active bool, width int) string {
	display := string(f.value)
	if f.kind == DialogFieldPassword {
		display = strings.Repeat("•", len([]rune(f.value)))
	}

	if active {
		display += "▏"
	} else if display == "" {
		display = " "
	}

	style := dialogFieldStyle
	if active {
		style = dialogFieldActiveStyle
	}
	return style.Width(width).Render(display)
}
