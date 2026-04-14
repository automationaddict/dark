package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/services/users"
)

type usersRequest struct {
	Username    string `json:"username,omitempty"`
	FullName    string `json:"full_name,omitempty"`
	Shell       string `json:"shell,omitempty"`
	Group       string `json:"group,omitempty"`
	Password    string `json:"password,omitempty"`
	CurrentPass string `json:"current_pass,omitempty"`
	Admin       *bool  `json:"admin,omitempty"`
	RemoveHome  *bool  `json:"remove_home,omitempty"`
}

type usersResponse struct {
	Snapshot users.Snapshot `json:"snapshot"`
	Error    string        `json:"error,omitempty"`
}

func wireUsers(nc *nats.Conn, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectUsersSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(users.ReadSnapshot())
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectUsersSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(usersRequest) usersResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req usersRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := usersResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				respond(m, data)
				return
			}
			resp := handler(req)
			data, _ := json.Marshal(resp)
			respond(m, data)
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	register(bus.SubjectUsersAddCmd, func(req usersRequest) usersResponse {
		if req.Username == "" {
			return usersResponse{Error: "missing username"}
		}
		admin := req.Admin != nil && *req.Admin
		if err := users.AddUser(req.Username, req.FullName, req.Shell, admin); err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersRemoveCmd, func(req usersRequest) usersResponse {
		if req.Username == "" {
			return usersResponse{Error: "missing username"}
		}
		removeHome := req.RemoveHome != nil && *req.RemoveHome
		if err := users.RemoveUser(req.Username, removeHome); err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersShellCmd, func(req usersRequest) usersResponse {
		if req.Username == "" || req.Shell == "" {
			return usersResponse{Error: "missing username or shell"}
		}
		if err := users.SetShell(req.Username, req.Shell); err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersCommentCmd, func(req usersRequest) usersResponse {
		if req.Username == "" {
			return usersResponse{Error: "missing username"}
		}
		if err := users.SetFullName(req.Username, req.FullName); err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersLockCmd, func(req usersRequest) usersResponse {
		if req.Username == "" {
			return usersResponse{Error: "missing username"}
		}
		lock := req.Admin == nil || !*req.Admin // admin=false means lock, admin=true means unlock
		var err error
		if lock {
			err = users.LockUser(req.Username)
		} else {
			err = users.UnlockUser(req.Username)
		}
		if err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersGroupCmd, func(req usersRequest) usersResponse {
		if req.Username == "" || req.Group == "" {
			return usersResponse{Error: "missing username or group"}
		}
		add := req.Admin == nil || *req.Admin // admin=true means add, admin=false means remove
		var err error
		if add {
			err = users.AddToGroup(req.Username, req.Group)
		} else {
			err = users.RemoveFromGroup(req.Username, req.Group)
		}
		if err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersAdminCmd, func(req usersRequest) usersResponse {
		if req.Username == "" {
			return usersResponse{Error: "missing username"}
		}
		admin := req.Admin != nil && *req.Admin
		var err error
		if admin {
			err = users.AddToGroup(req.Username, "wheel")
		} else {
			err = users.RemoveFromGroup(req.Username, "wheel")
		}
		if err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersPasswdCmd, func(req usersRequest) usersResponse {
		if req.Username == "" || req.Password == "" {
			return usersResponse{Error: "missing username or password"}
		}
		if err := users.SetPassword(req.Username, req.CurrentPass, req.Password); err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: users.ReadSnapshot()}
	})

	register(bus.SubjectUsersElevateCmd, func(req usersRequest) usersResponse {
		snap, err := users.ElevatedSnapshot()
		if err != nil {
			return usersResponse{Error: err.Error()}
		}
		return usersResponse{Snapshot: snap}
	})

	publish := func() {
		data, err := json.Marshal(users.ReadSnapshot())
		if err != nil {
			dn.Error("Users", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectUsersSnapshot, data); err != nil {
			dn.Error("Users", "publish failed: "+err.Error())
		}
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			publish()
		}
	}()

	return publish
}
