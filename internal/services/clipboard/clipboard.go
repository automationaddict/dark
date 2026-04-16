// Package clipboard is a minimal wrapper around the host's clipboard
// tool (wl-copy on Wayland, xclip on X11). Dark uses this to put the
// public half of SSH keys on the clipboard so users can paste them
// into GitHub / GitLab / their servers without opening ~/.ssh.
//
// The package has exactly one public function: Copy. Detection is
// lazy — we probe PATH on each call because the user may install the
// right tool while dark is running.
package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Copy writes s to the system clipboard using wl-copy (preferred) or
// xclip (fallback). Returns an error when neither tool is available
// or when the write fails. The caller is expected to surface the
// error to the user — dark never silently drops clipboard failures
// because the user's mental model is "I pressed copy, it's in my
// paste buffer."
func Copy(s string) error {
	if path, err := exec.LookPath("wl-copy"); err == nil {
		return run(path, nil, s)
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		return run(path, []string{"-selection", "clipboard"}, s)
	}
	return fmt.Errorf("no clipboard tool found — install wl-clipboard or xclip")
}

func run(path string, args []string, stdin string) error {
	cmd := exec.Command(path, args...)
	cmd.Stdin = bytes.NewBufferString(stdin)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", path, errBuf.String())
	}
	return nil
}
