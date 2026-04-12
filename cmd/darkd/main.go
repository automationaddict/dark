package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/lock"
	"github.com/johnnelson/dark/internal/logging"
	"github.com/johnnelson/dark/internal/scripting"
	appstoresvc "github.com/johnnelson/dark/internal/services/appstore"
	audiosvc "github.com/johnnelson/dark/internal/services/audio"
	btsvc "github.com/johnnelson/dark/internal/services/bluetooth"
	netsvc "github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/sysinfo"
	wifisvc "github.com/johnnelson/dark/internal/services/wifi"
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

type heartbeatMsg struct {
	Time    time.Time `json:"time"`
	Seq     uint64    `json:"seq"`
	Version string    `json:"version"`
}

func main() {
	logging.Setup("darkd")

	lk, err := lock.Acquire("darkd")
	if err != nil {
		fmt.Fprintln(os.Stderr, "darkd:", err)
		os.Exit(1)
	}
	defer lk.Release()

	srv, nc, err := bus.StartServer()
	if err != nil {
		fmt.Fprintln(os.Stderr, "darkd:", err)
		os.Exit(1)
	}
	defer func() {
		nc.Drain()
		srv.Shutdown()
		bus.CleanupDiscoveryFile()
	}()

	binPath, err := os.Executable()
	if err != nil {
		binPath = os.Args[0]
	}

	dn := newDaemonNotifier()
	defer dn.Close()

	wifiService, err := wifisvc.NewService()
	if err != nil {
		dn.Warn("Wi-Fi", fmt.Sprintf("%v — falling back to sysfs-only detection", err))
	}
	if wifiService != nil {
		defer wifiService.Close()
		if err := wifiService.StartAgent(); err != nil {
			dn.Warn("Wi-Fi", fmt.Sprintf("iwd agent failed: %v — unknown-network connects will fail", err))
		} else {
			slog.Info("agent registered", "service", "wifi", "agent", "iwd")
		}
	}

	bluetoothService, err := btsvc.NewService()
	if err != nil {
		dn.Warn("Bluetooth", fmt.Sprintf("%v — bluetooth controls unavailable", err))
	}
	if bluetoothService != nil {
		defer bluetoothService.Close()
		if err := bluetoothService.StartAgent(); err != nil {
			dn.Warn("Bluetooth", fmt.Sprintf("bluez agent failed: %v — pairing prompts will not be handled", err))
		} else {
			slog.Info("agent registered", "service", "bluetooth", "agent", "bluez")
		}
	}

	audioService, err := audiosvc.NewService()
	if err != nil {
		dn.Warn("Sound", fmt.Sprintf("%v — sound controls unavailable", err))
	}
	if audioService != nil {
		defer audioService.Close()
		slog.Info("service connected", "service", "audio", "backend", "pulse")
	}

	networkService, err := netsvc.NewService()
	if err != nil {
		dn.Warn("Network", fmt.Sprintf("%v — network backend unavailable", err))
	}
	if networkService != nil {
		defer networkService.Close()
	}

	appstoreLog := appstoreLogger()
	scriptEngine := scripting.New(appstoreLog)
	defer scriptEngine.Close()
	appstoreService := appstoresvc.NewService(appstoreLog, scriptEngine)
	defer appstoreService.Close()
	appstoreLog.Info("appstore: service ready")

	slog.Info("listening", "url", srv.ClientURL(), "discovery", bus.DiscoveryPath())

	// Reply handler for on-demand sysinfo requests so a freshly launched
	// TUI doesn't have to wait for the next periodic publish to render.
	if _, err := nc.Subscribe(bus.SubjectSystemInfoCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(sysinfo.Gather(binPath))
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectSystemInfoCmd, "error", err); os.Exit(1)
	}

	// Wifi snapshot responder. We reuse the long-lived Service built at
	// startup so every poll reuses the same D-Bus connection.
	if _, err := nc.Subscribe(bus.SubjectWifiAdaptersCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotWifi(wifiService))
		_ = m.Respond(data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectWifiAdaptersCmd, "error", err); os.Exit(1)
	}

	// Wifi action responders. Each one parses a request, delegates to a
	// handler that talks to the wifi service, then broadcasts the
	// refreshed snapshot so every connected TUI updates in lockstep.
	registerWifiAction := func(subject string, handler func(*wifisvc.Service, wifiActionRequest) wifiActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req wifiActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := wifiActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				_ = m.Respond(data)
				return
			}
			resp := handler(wifiService, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Wi-Fi", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectWifiAdapters, snapData); err != nil {
					dn.Error("Wi-Fi", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err); os.Exit(1)
		}
	}

	registerWifiAction(bus.SubjectWifiScanCmd, func(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
		return handleScan(svc, req.Adapter)
	})
	registerWifiAction(bus.SubjectWifiConnectCmd, handleConnect)
	registerWifiAction(bus.SubjectWifiDisconnectCmd, handleDisconnect)
	registerWifiAction(bus.SubjectWifiForgetCmd, handleForget)
	registerWifiAction(bus.SubjectWifiPowerCmd, handlePower)
	registerWifiAction(bus.SubjectWifiAutoconnectCmd, handleAutoconnect)
	registerWifiAction(bus.SubjectWifiConnectHiddenCmd, handleConnectHidden)
	registerWifiAction(bus.SubjectWifiAPStartCmd, handleAPStart)
	registerWifiAction(bus.SubjectWifiAPStopCmd, handleAPStop)

	publishBluetooth := wireBluetooth(nc, bluetoothService, dn)
	publishAudio := wireAudio(nc, audioService, dn)
	publishNetwork := wireNetwork(nc, networkService, dn)
	publishAppstore := wireAppstore(nc, appstoreService, appstoreLog, dn)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	heartbeat := time.NewTicker(core.TickHeartbeat)
	defer heartbeat.Stop()

	sysTick := time.NewTicker(core.TickSysInfo)
	defer sysTick.Stop()

	wifiTick := time.NewTicker(core.TickWifi)
	defer wifiTick.Stop()

	bluetoothTick := time.NewTicker(core.TickBluetooth)
	defer bluetoothTick.Stop()

	audioTick := time.NewTicker(core.TickAudio)
	defer audioTick.Stop()

	networkTick := time.NewTicker(core.TickNetwork)
	defer networkTick.Stop()

	appstoreTick := time.NewTicker(core.TickAppstore)
	defer appstoreTick.Stop()

	// Publish initial snapshots immediately so any subscriber that
	// connects in the gap before the first tick still gets pushed data.
	publishSysInfo(nc, binPath, dn)
	publishWifi(nc, wifiService, dn)
	publishBluetooth()
	publishAudio()
	publishNetwork()
	publishAppstore()

	var seq uint64
	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGHUP {
				slog.Info("reloading (SIGHUP)")
				publishSysInfo(nc, binPath, dn)
				publishWifi(nc, wifiService, dn)
				publishBluetooth()
				publishAudio()
				publishNetwork()
				publishAppstore()
				continue
			}
			slog.Info("shutting down", "signal", sig.String())
			go func() {
				time.Sleep(core.ShutdownTimeout)
				slog.Error("shutdown timeout — force exit")
				os.Exit(1)
			}()
			return
		case t := <-heartbeat.C:
			seq++
			data, _ := json.Marshal(heartbeatMsg{
				Time:    t,
				Seq:     seq,
				Version: "0.1.0-dev",
			})
			_ = nc.Publish(bus.SubjectDaemonHeartbeat, data)
		case <-sysTick.C:
			publishSysInfo(nc, binPath, dn)
		case <-wifiTick.C:
			publishWifi(nc, wifiService, dn)
		case <-bluetoothTick.C:
			publishBluetooth()
		case <-audioTick.C:
			publishAudio()
		case <-networkTick.C:
			publishNetwork()
		case <-appstoreTick.C:
			publishAppstore()
		}
	}
}

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

func publishSysInfo(nc *nats.Conn, binPath string, dn *daemonNotifier) {
	data, err := json.Marshal(sysinfo.Gather(binPath))
	if err != nil {
		dn.Error("System", "marshal failed: "+err.Error())
		return
	}
	if err := nc.Publish(bus.SubjectSystemInfo, data); err != nil {
		dn.Error("System", "publish failed: "+err.Error())
	}
}

func publishWifi(nc *nats.Conn, svc *wifisvc.Service, dn *daemonNotifier) {
	data, err := json.Marshal(snapshotWifi(svc))
	if err != nil {
		dn.Error("Wi-Fi", "marshal failed: "+err.Error())
		return
	}
	if err := nc.Publish(bus.SubjectWifiAdapters, data); err != nil {
		dn.Error("Wi-Fi", "publish failed: "+err.Error())
	}
}
