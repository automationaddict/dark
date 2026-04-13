package main

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/wifi"
	"github.com/johnnelson/dark/internal/tui"
)

// wifiConnectRequest is a wifiRequest variant that also carries a
// passphrase. Used for credentialed connect and hidden-network connect.
func wifiConnectRequest(nc *nats.Conn, subject, adapter, ssid, passphrase string) tui.WifiActionResultMsg {
	payload, _ := json.Marshal(map[string]string{
		"adapter":    adapter,
		"ssid":       ssid,
		"passphrase": passphrase,
	})
	reply, err := nc.Request(subject, payload, core.TimeoutLong)
	if err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot wifi.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.WifiActionResultMsg{Err: resp.Error}
	}
	return tui.WifiActionResultMsg{Snapshot: resp.Snapshot}
}

// wifiBoolRequest is a wifiRequest variant that also carries a bool flag
// in the payload. Used for power toggle and autoconnect toggle.
func wifiBoolRequest(nc *nats.Conn, subject, adapter, ssid string, flag bool) tui.WifiActionResultMsg {
	payload, _ := json.Marshal(map[string]any{
		"adapter": adapter,
		"ssid":    ssid,
		"powered": flag,
	})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot wifi.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.WifiActionResultMsg{Err: resp.Error}
	}
	return tui.WifiActionResultMsg{Snapshot: resp.Snapshot}
}

// wifiRequest sends a wifi action request to darkd and returns the
// refreshed snapshot from the reply. Used by every action command.
func wifiRequest(nc *nats.Conn, subject, adapter, ssid string) (wifi.Snapshot, error) {
	payload, _ := json.Marshal(map[string]string{"adapter": adapter, "ssid": ssid})
	reply, err := nc.Request(subject, payload, core.TimeoutConnect)
	if err != nil {
		return wifi.Snapshot{}, err
	}
	var resp struct {
		Snapshot wifi.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return wifi.Snapshot{}, err
	}
	if resp.Error != "" {
		return wifi.Snapshot{}, fmt.Errorf("%s", resp.Error)
	}
	return resp.Snapshot, nil
}
