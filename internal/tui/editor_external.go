package tui

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// externalEditKind tags a running external-editor session so the
// completion handler in Model.Update knows which downstream bus
// command to dispatch once the user exits the editor.
type externalEditKind int

const (
	editKindScript externalEditKind = iota
	editKindTopbarConfig
	editKindTopbarStyle
	editKindScreensaverContent
)

// externalEditDoneMsg is delivered when the suspended editor
// finishes. TempPath is set when the content was written to a
// scratch file (topbar, screensaver); it's empty when the editor
// ran directly on a real on-disk path (scripts) and the file is
// already in its final location.
type externalEditDoneMsg struct {
	kind     externalEditKind
	name     string
	tempPath string
	err      error
}

// resolveEditor returns the command users want for external
// editing. $EDITOR wins; otherwise we look up nvim, vim, nano in
// that order so most machines hit a sane default without config.
func resolveEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	for _, candidate := range []string{"nvim", "vim", "nano"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate
		}
	}
	return "vi"
}

// editExistingFile hands off to $EDITOR on an absolute path and
// suspends bubbletea until the editor exits. The editor edits the
// real file in place, so plugins, undo history, and filetype
// detection all behave correctly. The done callback delivers an
// externalEditDoneMsg back into Model.Update.
func editExistingFile(kind externalEditKind, name, path string) tea.Cmd {
	cmd := exec.Command(resolveEditor(), path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return externalEditDoneMsg{kind: kind, name: name, err: err}
	})
}

// editEphemeralContent writes initial to a temp file with a sensible
// extension, runs $EDITOR on it, and returns an externalEditDoneMsg
// carrying the temp path so Model.Update can read the updated
// content and dispatch a bus save. The temp file is intentionally
// left around until Update runs — deleting it inside the callback
// would race with the Update handler.
func editEphemeralContent(kind externalEditKind, name, ext, initial string) tea.Cmd {
	pattern := fmt.Sprintf("dark-%s-*%s", name, ext)
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return func() tea.Msg {
			return externalEditDoneMsg{kind: kind, name: name, err: err}
		}
	}
	path := f.Name()
	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		os.Remove(path)
		return func() tea.Msg {
			return externalEditDoneMsg{kind: kind, name: name, err: err}
		}
	}
	f.Close()
	cmd := exec.Command(resolveEditor(), path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return externalEditDoneMsg{kind: kind, name: name, tempPath: path, err: err}
	})
}

// handleExternalEditDone is the Model.Update side of the external-
// editor lifecycle. It reads the temp file when the edit used one,
// dispatches the kind-specific save (or a reload for scripts), and
// always cleans up the temp file on the way out.
func (m *Model) handleExternalEditDone(msg externalEditDoneMsg) tea.Cmd {
	if msg.tempPath != "" {
		defer os.Remove(msg.tempPath)
	}
	if msg.err != nil {
		slog.Warn("external editor failed", "kind", msg.kind, "error", msg.err.Error())
		m.notifyError("Editor", msg.err.Error())
		return nil
	}

	switch msg.kind {
	case editKindScript:
		// nvim wrote the file in place. Ask darkd to reload so any
		// hooks the script declares re-register without a restart.
		if m.scripting.ReloadScripts == nil {
			return nil
		}
		return m.scripting.ReloadScripts()

	case editKindTopbarConfig:
		content, err := os.ReadFile(msg.tempPath)
		if err != nil {
			m.notifyError("Top Bar", err.Error())
			return nil
		}
		if m.topbar.SetConfig == nil {
			return nil
		}
		m.state.TopBarBusy = true
		m.state.TopBarActionError = ""
		return m.topbar.SetConfig(string(content))

	case editKindTopbarStyle:
		content, err := os.ReadFile(msg.tempPath)
		if err != nil {
			m.notifyError("Top Bar", err.Error())
			return nil
		}
		if m.topbar.SetStyle == nil {
			return nil
		}
		m.state.TopBarBusy = true
		m.state.TopBarActionError = ""
		return m.topbar.SetStyle(string(content))

	case editKindScreensaverContent:
		content, err := os.ReadFile(msg.tempPath)
		if err != nil {
			m.notifyError("Screensaver", err.Error())
			return nil
		}
		if m.screensaver.SetContent == nil {
			return nil
		}
		m.state.ScreensaverBusy = true
		m.state.ScreensaverActionError = ""
		return m.screensaver.SetContent(string(content))
	}
	return nil
}
