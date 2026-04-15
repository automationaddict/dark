package main

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	liminesvc "github.com/automationaddict/dark/internal/services/limine"
)

type limineActionRequest struct {
	Description string   `json:"description,omitempty"`
	Number      int      `json:"number,omitempty"`
	Entry       int      `json:"entry,omitempty"`
	Key         string   `json:"key,omitempty"`
	Value       string   `json:"value,omitempty"`
	Lines       []string `json:"lines,omitempty"`
}

type limineActionResponse struct {
	Snapshot liminesvc.Snapshot `json:"snapshot"`
	Error    string             `json:"error,omitempty"`
}

// wireLimine registers all limine NATS handlers and returns a publisher
// closure the daemon's reload paths invoke. There is no periodic ticker
// for limine — snapshot state only changes when the user explicitly
// triggers a create/sync/delete action, so each action handler
// republishes after a successful run.
func wireLimine(nc *nats.Conn, svc *liminesvc.Service, dn *daemonNotifier) func() {
	if _, err := nc.Subscribe(bus.SubjectLimineSnapshotCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotLimine(svc))
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectLimineSnapshotCmd, "error", err)
		os.Exit(1)
	}

	register := func(subject string, handler func(*liminesvc.Service, limineActionRequest) limineActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req limineActionRequest
			if len(m.Data) > 0 {
				if err := json.Unmarshal(m.Data, &req); err != nil {
					resp := limineActionResponse{Error: "malformed request: " + err.Error()}
					data, _ := json.Marshal(resp)
					respond(m, data)
					return
				}
			}
			resp := handler(svc, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Limine", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectLimineSnapshot, snapData); err != nil {
					dn.Error("Limine", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err)
			os.Exit(1)
		}
	}

	register(bus.SubjectLimineCreateCmd, handleLimineCreate)
	register(bus.SubjectLimineDeleteCmd, handleLimineDelete)
	register(bus.SubjectLimineSyncCmd, handleLimineSync)
	register(bus.SubjectLimineDefaultEntryCmd, handleLimineDefaultEntry)
	register(bus.SubjectLimineBootConfigCmd, handleLimineBootConfig)
	register(bus.SubjectLimineSyncConfigCmd, handleLimineSyncConfig)
	register(bus.SubjectLimineOmarchyConfigCmd, handleLimineOmarchyConfig)
	register(bus.SubjectLimineKernelCmdlineCmd, handleLimineKernelCmdline)

	publish := func() {
		data, err := json.Marshal(snapshotLimine(svc))
		if err != nil {
			dn.Error("Limine", "marshal failed: "+err.Error())
			return
		}
		if err := nc.Publish(bus.SubjectLimineSnapshot, data); err != nil {
			dn.Error("Limine", "publish failed: "+err.Error())
		}
	}
	return publish
}

func snapshotLimine(svc *liminesvc.Service) liminesvc.Snapshot {
	if svc != nil {
		return svc.Snapshot()
	}
	return liminesvc.Detect()
}

func handleLimineCreate(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if err := svc.CreateSnapshot(req.Description); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	if err := svc.Sync(); err != nil {
		return limineActionResponse{Snapshot: svc.Snapshot(), Error: "snapshot created but sync failed: " + err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineDelete(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if err := svc.DeleteSnapshot(req.Number); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineSync(svc *liminesvc.Service, _ limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if err := svc.Sync(); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineDefaultEntry(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if err := svc.SetDefaultEntry(req.Entry); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineBootConfig(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if req.Key == "" {
		return limineActionResponse{Error: "missing key"}
	}
	if err := svc.SetBootConfig(req.Key, req.Value); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineSyncConfig(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if req.Key == "" {
		return limineActionResponse{Error: "missing key"}
	}
	if err := svc.SetSyncConfig(req.Key, req.Value); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineOmarchyConfig(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if req.Key == "" {
		return limineActionResponse{Error: "missing key"}
	}
	if err := svc.SetOmarchyConfig(req.Key, req.Value); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}

func handleLimineKernelCmdline(svc *liminesvc.Service, req limineActionRequest) limineActionResponse {
	if svc == nil {
		return limineActionResponse{Error: "limine service unavailable"}
	}
	if err := svc.SetOmarchyKernelCmdline(req.Lines); err != nil {
		return limineActionResponse{Error: err.Error()}
	}
	return limineActionResponse{Snapshot: svc.Snapshot()}
}
