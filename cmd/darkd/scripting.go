package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/scripting"
)

// wireScripting subscribes the three read-only Scripting handlers
// that back the F5 Scripting tab: user-script listing, Lua host
// registry, and enumerated dark.cmd.* API catalog. No publish loop —
// the UI pulls on demand the first time it enters F5.
// reloadUserScripts wipes existing user hooks and re-runs every
// .lua file in the user scripts directory. Called after a save or
// delete so the daemon's event dispatcher stays in sync with what
// the user sees in the F5 Scripting tab — no darkd restart needed.
func reloadUserScripts(engine *scripting.Engine) {
	if engine == nil {
		return
	}
	engine.ClearUserHooks()
	scripting.LoadAllUserScripts(engine)
}

func wireScripting(nc *nats.Conn, engine *scripting.Engine) {
	if _, err := nc.Subscribe(bus.SubjectScriptingListCmd, func(m *nats.Msg) {
		scripts, err := scripting.ListUserScripts()
		resp := scriptingListResponse{Scripts: scripts}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingListCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScriptingRegistryCmd, func(m *nats.Msg) {
		var entries []scripting.RegistryEntry
		if engine != nil {
			entries = engine.Registry().Entries()
		}
		data, _ := json.Marshal(scriptingRegistryResponse{Entries: entries})
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingRegistryCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScriptingAPICatalogCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(scriptingAPICatalogResponse{Commands: bus.APICommandCatalog()})
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingAPICatalogCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScriptingReadCmd, func(m *nats.Msg) {
		var req scriptingNameRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			data, _ := json.Marshal(scriptingReadResponse{Error: "malformed request: " + err.Error()})
			respond(m, data)
			return
		}
		content, err := scripting.ReadUserScript(req.Name)
		resp := scriptingReadResponse{Name: req.Name, Content: content}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingReadCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScriptingSaveCmd, func(m *nats.Msg) {
		var req scriptingSaveRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			data, _ := json.Marshal(scriptingWriteResponse{Error: "malformed request: " + err.Error()})
			respond(m, data)
			return
		}
		resp := scriptingWriteResponse{Name: req.Name}
		if err := scripting.SaveUserScript(req.Name, req.Content); err != nil {
			resp.Error = err.Error()
		} else {
			reloadUserScripts(engine)
			scripts, err := scripting.ListUserScripts()
			if err != nil {
				resp.Error = err.Error()
			} else {
				resp.Scripts = scripts
			}
		}
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingSaveCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScriptingCallCmd, func(m *nats.Msg) {
		var req scriptingCallRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			data, _ := json.Marshal(scriptingCallResponse{Error: "malformed request: " + err.Error()})
			respond(m, data)
			return
		}
		if req.Fn == "" {
			data, _ := json.Marshal(scriptingCallResponse{Error: "missing fn"})
			respond(m, data)
			return
		}
		result, err := engine.CallUserFunction(req.Fn, req.Args...)
		resp := scriptingCallResponse{Fn: req.Fn, Result: result}
		if err != nil {
			resp.Error = err.Error()
		}
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingCallCmd, "error", err)
		os.Exit(1)
	}

	if _, err := nc.Subscribe(bus.SubjectScriptingDeleteCmd, func(m *nats.Msg) {
		var req scriptingNameRequest
		if err := json.Unmarshal(m.Data, &req); err != nil {
			data, _ := json.Marshal(scriptingWriteResponse{Error: "malformed request: " + err.Error()})
			respond(m, data)
			return
		}
		resp := scriptingWriteResponse{Name: req.Name}
		if err := scripting.DeleteUserScript(req.Name); err != nil {
			resp.Error = err.Error()
		} else {
			reloadUserScripts(engine)
			scripts, err := scripting.ListUserScripts()
			if err != nil {
				resp.Error = err.Error()
			} else {
				resp.Scripts = scripts
			}
		}
		data, _ := json.Marshal(resp)
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectScriptingDeleteCmd, "error", err)
		os.Exit(1)
	}
}

type scriptingNameRequest struct {
	Name string `json:"name"`
}

type scriptingSaveRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type scriptingReadResponse struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

type scriptingWriteResponse struct {
	Name    string                 `json:"name"`
	Scripts []scripting.ScriptFile `json:"scripts,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

type scriptingCallRequest struct {
	Fn   string        `json:"fn"`
	Args []interface{} `json:"args,omitempty"`
}

type scriptingCallResponse struct {
	Fn     string      `json:"fn"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type scriptingListResponse struct {
	Scripts []scripting.ScriptFile `json:"scripts"`
	Error   string                 `json:"error,omitempty"`
}

type scriptingRegistryResponse struct {
	Entries []scripting.RegistryEntry `json:"entries"`
	Error   string                    `json:"error,omitempty"`
}

type scriptingAPICatalogResponse struct {
	Commands []bus.APICommandEntry `json:"commands"`
	Error    string                `json:"error,omitempty"`
}
