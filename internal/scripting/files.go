package scripting

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// ErrInvalidScriptName is returned when a script name contains path
// separators, leading dots, or doesn't end in .lua. The user scripts
// directory is flat by design — sub-directories are not supported.
var ErrInvalidScriptName = errors.New("invalid script name")

// ScriptFile is a filesystem record for a single user-editable Lua
// script. Mirrors core.ScriptEntry but lives in this package to keep
// the scripting layer free of upstream imports.
type ScriptFile struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Source    string    `json:"source"`
	SizeBytes int64     `json:"size_bytes"`
	ModTime   time.Time `json:"mod_time"`
	Preview   string    `json:"preview,omitempty"`
}

// UserScriptsDir returns the user scripts directory
// ($XDG_CONFIG_HOME/dark/scripts/) and ensures it exists. An empty
// string is returned when neither XDG_CONFIG_HOME nor HOME is usable.
func UserScriptsDir() string {
	dir := userScriptDir()
	if dir == "" {
		return ""
	}
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// ListUserScripts walks the user scripts directory and returns every
// .lua file as a ScriptFile. Directories and non-Lua files are
// skipped. Results are sorted by filename so the UI ordering is
// stable across runs.
func ListUserScripts() ([]ScriptFile, error) {
	dir := UserScriptsDir()
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]ScriptFile, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(dir, e.Name())
		out = append(out, ScriptFile{
			Name:      e.Name(),
			Path:      full,
			Source:    "user",
			SizeBytes: info.Size(),
			ModTime:   info.ModTime(),
			Preview:   readPreview(full),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// LoadUserScriptFile compiles and executes a script from an absolute
// path, bypassing the load-once cache used by LoadScript so reloads
// re-register hooks declared at script top level.
func (e *Engine) LoadUserScriptFile(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("scripting: read %s: %w", path, err)
	}
	fn, err := e.vm.LoadString(string(b))
	if err != nil {
		return fmt.Errorf("scripting: compile %s: %w", path, err)
	}
	e.vm.Push(fn)
	if err := e.vm.PCall(0, lua.MultRet, nil); err != nil {
		return fmt.Errorf("scripting: exec %s: %w", path, err)
	}
	e.logger.Info("scripting: user script loaded", "path", path)
	return nil
}

// SeedExampleScripts copies the built-in example scripts into the
// user scripts directory on first run. Existing files are never
// overwritten, so users can freely edit an example without having
// it stomped on the next daemon restart. Removing a seeded example
// is also sticky — once deleted, it stays gone until the user drops
// a file of the same name back into the directory.
//
// The source lives under internal/scripting/scripts/examples/ and
// is compiled into the binary via the same //go:embed directive
// that powers the appstore category scripts.
func SeedExampleScripts(e *Engine) {
	if e == nil {
		return
	}
	dir := UserScriptsDir()
	if dir == "" {
		return
	}
	entries, err := defaultScripts.ReadDir("scripts/examples")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		target := filepath.Join(dir, name)
		if _, err := os.Stat(target); err == nil {
			continue
		}
		data, err := defaultScripts.ReadFile("scripts/examples/" + name)
		if err != nil {
			continue
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			e.logger.Warn("scripting: seed example failed",
				"name", name, "error", err.Error())
			continue
		}
		e.logger.Info("scripting: seeded example script", "name", name)
	}
}

// LoadAllUserScripts compiles and executes every .lua file in the
// user scripts directory, in lexical order, so their top-level
// `dark.on(...)` calls register hooks on the engine. Errors on
// individual scripts are logged through the engine logger but do
// not abort the batch — a single broken script must not keep the
// rest of the user's Lua code from running.
func LoadAllUserScripts(e *Engine) {
	if e == nil {
		return
	}
	list, err := ListUserScripts()
	if err != nil {
		e.logger.Warn("scripting: list user scripts failed", "error", err.Error())
		return
	}
	for _, f := range list {
		if err := e.LoadUserScriptFile(f.Path); err != nil {
			e.logger.Warn("scripting: load user script failed",
				"path", f.Path, "error", err.Error())
		}
	}
}

// ReadUserScript returns the full contents of a script by its base
// name. Returns ErrInvalidScriptName for traversal attempts or names
// that don't look like a flat .lua file.
func ReadUserScript(name string) (string, error) {
	path, err := resolveScriptPath(name)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SaveUserScript writes content to the named script file, creating
// it if necessary. The user scripts directory is created on first
// write. Rejects invalid names to prevent path traversal.
func SaveUserScript(name, content string) error {
	path, err := resolveScriptPath(name)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// DeleteUserScript removes a script file from the user scripts
// directory. Missing files are reported as an error — the caller
// can decide to treat ENOENT as success.
func DeleteUserScript(name string) error {
	path, err := resolveScriptPath(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// resolveScriptPath validates name and returns its absolute path
// inside the user scripts directory. Also ensures the directory
// exists so a first-time save doesn't trip over ENOENT.
func resolveScriptPath(name string) (string, error) {
	if name == "" {
		return "", ErrInvalidScriptName
	}
	if name != filepath.Base(name) {
		return "", ErrInvalidScriptName
	}
	if strings.HasPrefix(name, ".") {
		return "", ErrInvalidScriptName
	}
	if !strings.HasSuffix(strings.ToLower(name), ".lua") {
		return "", ErrInvalidScriptName
	}
	dir := UserScriptsDir()
	if dir == "" {
		return "", errors.New("user scripts directory unavailable")
	}
	return filepath.Join(dir, name), nil
}

// readPreview returns the first ~2KB of a script file so the detail
// pane has something to show without the full editor. Errors become
// an empty string — the list view still renders path + stats.
func readPreview(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, 2048)
	n, _ := f.Read(buf)
	return string(buf[:n])
}
