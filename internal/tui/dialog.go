package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type DialogFieldKind int

const (
	DialogFieldText DialogFieldKind = iota
	DialogFieldPassword
	DialogFieldSelect
)

// DialogFieldSpec defines one form row. Declared up front when building
// the dialog so the Dialog struct can carry internal state per field.
type DialogFieldSpec struct {
	Key      string
	Label    string
	Kind     DialogFieldKind
	Value    string   // optional pre-filled value
	Options  []string // for DialogFieldSelect
	OnChange func(value string) // called when select cursor moves
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
	key       string
	label     string
	kind      DialogFieldKind
	value     []rune
	options   []string
	selectIdx int
	scrollOff int
	onChange  func(string)
}

// NewDialog constructs a dialog with the given title, field definitions,
// and submit callback.
func NewDialog(title string, fields []DialogFieldSpec, submit DialogSubmitFunc) *Dialog {
	d := &Dialog{title: title, submit: submit}
	for _, f := range fields {
		df := dialogField{
			key:      f.Key,
			label:    f.Label,
			kind:     f.Kind,
			value:    []rune(f.Value),
			options:  f.Options,
			onChange: f.OnChange,
		}
		if f.Kind == DialogFieldSelect {
			for i, o := range f.Options {
				if o == f.Value {
					df.selectIdx = i
					break
				}
			}
		}
		d.fields = append(d.fields, df)
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

	af := d.activeField()
	if af != nil && af.kind == DialogFieldSelect {
		return d.updateSelect(msg)
	}

	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		d.closed = true
		return nil
	case tea.KeyEnter:
		return d.doSubmit()
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

func (d *Dialog) updateSelect(msg tea.KeyMsg) tea.Cmd {
	af := d.activeField()
	if af == nil {
		return nil
	}
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		d.closed = true
		return nil
	case tea.KeyEnter:
		if len(af.options) > 0 {
			af.value = []rune(af.options[af.selectIdx])
		}
		return d.doSubmit()
	case tea.KeyTab:
		d.advanceField(1)
	case tea.KeyShiftTab:
		d.advanceField(-1)
	default:
		prev := af.selectIdx
		switch msg.String() {
		case "j", "down":
			if af.selectIdx < len(af.options)-1 {
				af.selectIdx++
			}
		case "k", "up":
			if af.selectIdx > 0 {
				af.selectIdx--
			}
		case "g", "home":
			af.selectIdx = 0
		case "G", "end":
			af.selectIdx = len(af.options) - 1
		}
		if af.selectIdx != prev && af.onChange != nil && len(af.options) > 0 {
			af.onChange(af.options[af.selectIdx])
		}
	}
	return nil
}

func (d *Dialog) doSubmit() tea.Cmd {
	values := DialogResult{}
	for _, f := range d.fields {
		if f.kind == DialogFieldSelect && len(f.options) > 0 {
			values[f.key] = f.options[f.selectIdx]
		} else {
			values[f.key] = string(f.value)
		}
	}
	d.closed = true
	if d.submit != nil {
		return d.submit(values)
	}
	return nil
}

func (d *Dialog) activeField() *dialogField {
	if len(d.fields) == 0 || d.active >= len(d.fields) {
		return nil
	}
	return &d.fields[d.active]
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
	if f.kind == DialogFieldSelect {
		return
	}
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
	if f.kind == DialogFieldSelect {
		return
	}
	f.value = append(f.value, runes...)
}

// View renders the dialog as a bordered modal box. Caller is responsible
// for compositing it onto the base view via overlayCenter.
func (d *Dialog) View() string {
	const innerWidth = 48

	var rows []string

	for i, f := range d.fields {
		label := dialogFieldLabelStyle.Render(f.label)
		if i == d.active {
			label = dialogFieldLabelActiveStyle.Render(f.label)
		}
		rows = append(rows, label)

		if f.kind == DialogFieldSelect {
			rows = append(rows, renderDialogSelect(f, i == d.active, innerWidth))
		} else {
			rows = append(rows, renderDialogField(f, i == d.active, innerWidth))
		}
		rows = append(rows, "")
	}

	hint := "enter submit · esc cancel"
	af := d.activeField()
	if af != nil && af.kind == DialogFieldSelect {
		hint = "j/k select · enter submit · esc cancel"
	} else if len(d.fields) > 1 {
		hint = "enter submit · tab next field · esc cancel"
	}
	rows = append(rows, dialogHintStyle.Render(hint))

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

const selectVisibleRows = 8

func renderDialogSelect(f dialogField, active bool, width int) string {
	if len(f.options) == 0 {
		return dialogFieldStyle.Width(width).Render("(no options)")
	}

	start := f.scrollOff
	if f.selectIdx < start {
		start = f.selectIdx
	}
	if f.selectIdx >= start+selectVisibleRows {
		start = f.selectIdx - selectVisibleRows + 1
	}
	if start < 0 {
		start = 0
	}

	end := start + selectVisibleRows
	if end > len(f.options) {
		end = len(f.options)
	}

	var lines []string
	for i := start; i < end; i++ {
		opt := f.options[i]
		if i == f.selectIdx && active {
			line := dialogFieldActiveStyle.Width(width).Render("▸ " + opt)
			lines = append(lines, line)
		} else {
			line := dialogFieldStyle.Width(width).Render("  " + opt)
			lines = append(lines, line)
		}
	}

	if start > 0 {
		lines = append([]string{dialogHintStyle.Render("  ↑ more")}, lines...)
	}
	if end < len(f.options) {
		lines = append(lines, dialogHintStyle.Render("  ↓ more"))
	}

	return strings.Join(lines, "\n")
}
