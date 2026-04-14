package limine

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// rewriteBootConfKey sets key: value on the first matching top-level
// line in /boot/limine.conf, uncommenting a commented-out form if it's
// present. Auto-generated /+entry blocks below are left untouched —
// top-level keys by convention live at the top of the file and
// limine-entry-tool only regenerates entries, not the header.
func rewriteBootConfKey(content, key, value string) string {
	lines := strings.Split(content, "\n")
	set := false
	inEntry := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/") {
			inEntry = true
		}
		if inEntry {
			continue
		}
		if set {
			continue
		}
		// Match "key:" or "#key:" (with optional leading whitespace).
		stripped := strings.TrimLeft(trimmed, "#")
		stripped = strings.TrimSpace(stripped)
		if !strings.HasPrefix(stripped, key+":") {
			continue
		}
		lines[i] = fmt.Sprintf("%s: %s", key, value)
		set = true
	}
	if set {
		return strings.Join(lines, "\n")
	}
	// Append before the first entry line if none found.
	insertAt := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "/") {
			insertAt = i
			break
		}
	}
	newLine := fmt.Sprintf("%s: %s", key, value)
	out := make([]string, 0, len(lines)+2)
	out = append(out, lines[:insertAt]...)
	// Keep a blank line separating appended keys from the entries
	// below for readability.
	if insertAt > 0 && strings.TrimSpace(lines[insertAt-1]) != "" {
		out = append(out, newLine, "")
	} else {
		out = append(out, newLine)
	}
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n")
}

// shellAssignmentRe matches KEY= or #KEY= at the start of a line
// (with optional leading whitespace). Used to find an existing
// assignment to replace.
var shellAssignmentRe = regexp.MustCompile(`^(#\s*)?([A-Z_][A-Z0-9_]*)\s*=`)

// rewriteShellAssignment sets KEY="value" on the first matching line
// in a shell-style config file, uncommenting a commented form if
// necessary. Appends to the end of the file if no match is found.
// Values are always quoted so callers don't have to worry about
// whitespace in the value.
func rewriteShellAssignment(content, key, value string) string {
	lines := strings.Split(content, "\n")
	newLine := fmt.Sprintf(`%s="%s"`, key, value)
	for i, line := range lines {
		m := shellAssignmentRe.FindStringSubmatch(line)
		if m == nil || m[2] != key {
			continue
		}
		lines[i] = newLine
		return strings.Join(lines, "\n")
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = append(lines[:len(lines)-1], newLine, "")
	} else {
		lines = append(lines, newLine)
	}
	return strings.Join(lines, "\n")
}

// rewriteShellCmdline replaces every `KERNEL_CMDLINE[default]+=` line
// in content with the supplied lines, preserving the position of the
// first match (so the cmdline block stays wherever it was). If no
// existing line is found, the new block is appended to the end of the
// file.
func rewriteShellCmdline(content string, values []string) string {
	lines := strings.Split(content, "\n")
	const prefix = "KERNEL_CMDLINE[default]+="
	firstIdx := -1
	var filtered []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			if firstIdx == -1 {
				firstIdx = len(filtered)
			}
			continue
		}
		filtered = append(filtered, line)
	}
	block := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		block = append(block, fmt.Sprintf(`%s"%s"`, prefix, v))
	}
	if firstIdx == -1 {
		firstIdx = len(filtered)
	}
	out := make([]string, 0, len(filtered)+len(block))
	out = append(out, filtered[:firstIdx]...)
	out = append(out, block...)
	out = append(out, filtered[firstIdx:]...)
	return strings.Join(out, "\n")
}

// readFile is a tiny convenience over os.ReadFile that returns empty
// content on a missing file so callers can still append a brand-new
// key to an otherwise-unmanaged file.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// writeWithPkexec stages content in /tmp and invokes
// `pkexec install -m 644 -o root -g root tmp dest` so the replacement
// is atomic and keeps the canonical file ownership/permissions. The
// tempfile is removed whether or not install succeeds.
func writeWithPkexec(dest, content string) error {
	tmp, err := os.CreateTemp("", "dark-limine-*.conf")
	if err != nil {
		return fmt.Errorf("stage tempfile: %w", err)
	}
	defer os.Remove(tmp.Name())
	w := bufio.NewWriter(tmp)
	if _, err := w.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write tempfile: %w", err)
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		return fmt.Errorf("flush tempfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tempfile: %w", err)
	}
	return runPkexec("install", "-m", "644", "-o", "root", "-g", "root",
		tmp.Name(), dest)
}

// setBootConfKey is the full pipeline: read the current
// /boot/limine.conf, rewrite the requested key, and install the
// result via pkexec.
func setBootConfKey(path, key, value string) error {
	content, err := readFile(path)
	if err != nil {
		return err
	}
	return writeWithPkexec(path, rewriteBootConfKey(content, key, value))
}

func setShellAssignment(path, key, value string) error {
	content, err := readFile(path)
	if err != nil {
		return err
	}
	return writeWithPkexec(path, rewriteShellAssignment(content, key, value))
}

func setShellCmdline(path string, values []string) error {
	content, err := readFile(path)
	if err != nil {
		return err
	}
	return writeWithPkexec(path, rewriteShellCmdline(content, values))
}
