package scripting

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

// This file owns the "stdlib" host functions that scripts need for
// real work: file I/O, process execution, XDG path lookups, and JSON
// encode/decode. Split out of darkmod.go so the core registration
// path stays focused on the event/action surface.
//
// All functions return two values — the result on the left and an
// error string on the right — so Lua callers can destructure with
// `local value, err = dark.read_file("...")` and branch on `err`.

// luaReadFile implements dark.read_file(path). Returns the file's
// contents as a string on success, or (nil, error) on failure.
func (e *Engine) luaReadFile(L *lua.LState) int {
	path := L.CheckString(1)
	b, err := os.ReadFile(path)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(string(b)))
	L.Push(lua.LNil)
	return 2
}

// luaWriteFile implements dark.write_file(path, content). Creates
// any missing parent directories. Returns (true, nil) on success
// or (false, error) on failure.
func (e *Engine) luaWriteFile(L *lua.LState) int {
	path := L.CheckString(1)
	content := L.CheckString(2)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	L.Push(lua.LNil)
	return 2
}

// luaRun implements dark.run(cmd, args?) — synchronous command
// execution with captured output. Returns a result table with
// `stdout`, `stderr`, and `code` fields, or (nil, error) when the
// command can't be started at all. A non-zero exit is returned as
// a successful result with a non-zero code — the caller decides
// whether that's an error.
func (e *Engine) luaRun(L *lua.LState) int {
	cmdName := L.CheckString(1)
	args := luaArrayToStrings(L, 2)

	cmd := exec.Command(cmdName, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	code := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			L.Push(lua.LNil)
			L.Push(lua.LString(runErr.Error()))
			return 2
		}
	}

	result := L.NewTable()
	result.RawSetString("stdout", lua.LString(outBuf.String()))
	result.RawSetString("stderr", lua.LString(errBuf.String()))
	result.RawSetString("code", lua.LNumber(code))
	L.Push(result)
	L.Push(lua.LNil)
	return 2
}

// luaSpawn implements dark.spawn(cmd, args?) — fire-and-forget
// process execution. Returns the child PID on success or
// (nil, error) on failure. The engine reaps the process in the
// background so it doesn't turn into a zombie.
func (e *Engine) luaSpawn(L *lua.LState) int {
	cmdName := L.CheckString(1)
	args := luaArrayToStrings(L, 2)

	cmd := exec.Command(cmdName, args...)
	if err := cmd.Start(); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	go func() {
		_ = cmd.Wait()
	}()
	L.Push(lua.LNumber(cmd.Process.Pid))
	L.Push(lua.LNil)
	return 2
}

// luaArrayToStrings reads the positional arg at idx as a Lua table
// of strings. Missing / nil arg returns nil. Non-table values and
// non-string elements become empty strings rather than raising.
func luaArrayToStrings(L *lua.LState, idx int) []string {
	if L.GetTop() < idx || L.Get(idx) == lua.LNil {
		return nil
	}
	tbl, ok := L.Get(idx).(*lua.LTable)
	if !ok {
		return nil
	}
	out := make([]string, 0, tbl.Len())
	tbl.ForEach(func(k, v lua.LValue) {
		if _, ok := k.(lua.LNumber); ok {
			out = append(out, v.String())
		}
	})
	return out
}

// luaHome implements dark.home() — returns the user's home
// directory, or an empty string when os.UserHomeDir fails.
func (e *Engine) luaHome(L *lua.LState) int {
	home, err := os.UserHomeDir()
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(home))
	return 1
}

// luaScriptsDir implements dark.scripts_dir() — returns the user
// scripts directory (~/.config/dark/scripts by default). Same path
// the daemon uses for loading scripts so `dark.write_file(path,...)`
// inside that dir can create files that auto-load on next reload.
func (e *Engine) luaScriptsDir(L *lua.LState) int {
	L.Push(lua.LString(userScriptDir()))
	return 1
}

// luaConfigDir implements dark.config_dir() — XDG_CONFIG_HOME,
// falling back to $HOME/.config.
func (e *Engine) luaConfigDir(L *lua.LState) int {
	L.Push(lua.LString(xdgDir("XDG_CONFIG_HOME", ".config")))
	return 1
}

// luaCacheDir implements dark.cache_dir() — XDG_CACHE_HOME,
// falling back to $HOME/.cache.
func (e *Engine) luaCacheDir(L *lua.LState) int {
	L.Push(lua.LString(xdgDir("XDG_CACHE_HOME", ".cache")))
	return 1
}

// xdgDir returns the value of the given XDG env var, or
// $HOME/<fallback> when the env var is unset.
func xdgDir(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, fallback)
}

// luaJSONEncode implements dark.json_encode(value). Returns the
// JSON string on success or (nil, error) on failure.
func (e *Engine) luaJSONEncode(L *lua.LState) int {
	v := luaValueToGo(L.Get(1))
	b, err := json.Marshal(v)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(string(b)))
	L.Push(lua.LNil)
	return 2
}

// luaJSONDecode implements dark.json_decode(string). Returns the
// decoded value as a Lua table / scalar on success or (nil, error)
// on failure.
func (e *Engine) luaJSONDecode(L *lua.LState) int {
	s := L.CheckString(1)
	var raw interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(goToLua(L, raw))
	L.Push(lua.LNil)
	return 2
}
