package scripting

import (
	"os"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// This file owns the small utility host functions attached to the
// dark Lua module: dark.notify, dark.env, dark.now, dark.hostname.
// They're one-liners around Go stdlib calls that scripts commonly
// need, split out of darkmod.go so the core registration path stays
// focused on dark.on / dark.log / dark.cmd / dark.actions.

// luaNotify implements dark.notify(summary, body, urgency?). Falls
// back to a log line when no notifier is installed so scripts still
// have a trail to follow on a headless daemon.
func (e *Engine) luaNotify(L *lua.LState) int {
	summary := L.CheckString(1)
	body := ""
	if L.GetTop() >= 2 {
		body = L.CheckString(2)
	}
	urgency := "normal"
	if L.GetTop() >= 3 {
		urgency = L.CheckString(3)
	}
	if e.notifier == nil {
		e.logger.Info("script: notify",
			"summary", summary, "body", body, "urgency", urgency)
		return 0
	}
	e.notifier(summary, body, urgency)
	return 0
}

// luaEnv implements dark.env(name). Returns the current value of
// the named environment variable or an empty string when it's unset.
// Read-only — scripts that want to mutate the environment should do
// it through dark.cmd / dark.spawn in a future revision.
func (e *Engine) luaEnv(L *lua.LState) int {
	name := L.CheckString(1)
	L.Push(lua.LString(os.Getenv(name)))
	return 1
}

// luaNow implements dark.now(). Returns the Unix timestamp as a
// Lua number so scripts can do debouncing or rate limiting without
// reaching for os.time / os.date.
func (e *Engine) luaNow(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().Unix()))
	return 1
}

// luaHostname implements dark.hostname(). Returns the host's
// network name, or an empty string when os.Hostname fails.
func (e *Engine) luaHostname(L *lua.LState) int {
	name, err := os.Hostname()
	if err != nil {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(name))
	return 1
}
