package main

import (
	"time"

	"github.com/automationaddict/dark/internal/core"
	wifisvc "github.com/automationaddict/dark/internal/services/wifi"
)

// wifi action protocol — shared by scan, connect, disconnect, forget.
// All action commands take an adapter name; connect/forget also take an
// ssid. Responses always carry the refreshed snapshot so the TUI can
// update its view in one shot.
type wifiActionRequest struct {
	Adapter    string `json:"adapter"`
	SSID       string `json:"ssid,omitempty"`
	Powered    *bool  `json:"powered,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

type wifiActionResponse struct {
	Snapshot wifisvc.Snapshot `json:"snapshot"`
	Error    string           `json:"error,omitempty"`
}

// Legacy aliases from Phase C — kept for handleScan's existing shape.
type scanRequest = wifiActionRequest
type scanResponse = wifiActionResponse

// snapshotWifi uses the long-lived iwd connection when available, or
// falls back to a one-shot Detect() call otherwise.
func snapshotWifi(svc *wifisvc.Service) wifisvc.Snapshot {
	if svc != nil {
		return svc.Snapshot()
	}
	return wifisvc.Detect()
}

// handleScan runs a live scan on the named adapter and returns the
// refreshed snapshot. Errors from iwd become typed error responses so the
// TUI can show them inline instead of failing the request silently.
func handleScan(svc *wifisvc.Service, adapter string) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if adapter == "" {
		return wifiActionResponse{Error: "missing adapter name"}
	}
	if err := svc.TriggerScan(adapter, 15*time.Second); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleConnect(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" || req.SSID == "" {
		return wifiActionResponse{Error: "missing adapter or ssid"}
	}
	err := svc.ConnectWithPassphrase(req.Adapter, req.SSID, req.Passphrase, 20*time.Second)
	if err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleConnectHidden(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" || req.SSID == "" {
		return wifiActionResponse{Error: "missing adapter or ssid"}
	}
	if err := svc.ConnectHidden(req.Adapter, req.SSID, req.Passphrase); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleAPStart(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" || req.SSID == "" {
		return wifiActionResponse{Error: "missing adapter or ssid"}
	}
	if err := svc.StartAP(req.Adapter, req.SSID, req.Passphrase); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	time.Sleep(core.IWDAPSnapshotWait)
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleAPStop(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" {
		return wifiActionResponse{Error: "missing adapter name"}
	}
	if err := svc.StopAP(req.Adapter); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	time.Sleep(core.IWDAPSnapshotWait)
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleDisconnect(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" {
		return wifiActionResponse{Error: "missing adapter name"}
	}
	if err := svc.Disconnect(req.Adapter); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleForget(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" || req.SSID == "" {
		return wifiActionResponse{Error: "missing adapter or ssid"}
	}
	if err := svc.Forget(req.Adapter, req.SSID); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handleAutoconnect(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.SSID == "" {
		return wifiActionResponse{Error: "missing ssid"}
	}
	if req.Powered == nil {
		return wifiActionResponse{Error: "missing autoconnect flag"}
	}
	if err := svc.SetAutoConnect(req.SSID, *req.Powered); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func handlePower(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
	if svc == nil {
		return wifiActionResponse{Error: "wifi service unavailable"}
	}
	if req.Adapter == "" {
		return wifiActionResponse{Error: "missing adapter name"}
	}
	if req.Powered == nil {
		return wifiActionResponse{Error: "missing powered flag"}
	}
	if err := svc.SetRadioPowered(req.Adapter, *req.Powered); err != nil {
		return wifiActionResponse{Error: err.Error()}
	}
	// Give iwd a moment to settle before reading back; Powered transitions
	// are fast but the downstream Device/Station state updates are async.
	time.Sleep(core.IWDPowerSettleWait)
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}
