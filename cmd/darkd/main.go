package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/lock"
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
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("darkd ")

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

	wifiService, err := wifisvc.NewService()
	if err != nil {
		log.Printf("wifi: %v (falling back to sysfs-only detection)", err)
	}
	if wifiService != nil {
		defer wifiService.Close()
		if err := wifiService.StartAgent(); err != nil {
			log.Printf("wifi: failed to register iwd agent: %v (unknown-network connects will fail)", err)
		} else {
			log.Printf("wifi: iwd agent registered")
		}
	}

	bluetoothService, err := btsvc.NewService()
	if err != nil {
		log.Printf("bluetooth: %v", err)
	}
	if bluetoothService != nil {
		defer bluetoothService.Close()
		if err := bluetoothService.StartAgent(); err != nil {
			log.Printf("bluetooth: failed to register bluez agent: %v (pairing prompts will not be handled)", err)
		} else {
			log.Printf("bluetooth: bluez agent registered")
		}
	}

	audioService, err := audiosvc.NewService()
	if err != nil {
		log.Printf("audio: %v (sound controls will be unavailable)", err)
	}
	if audioService != nil {
		defer audioService.Close()
		log.Printf("audio: pulse/pipewire-pulse client connected")
	}

	networkService, err := netsvc.NewService()
	if err != nil {
		log.Printf("network: %v", err)
	}
	if networkService != nil {
		defer networkService.Close()
	}

	appstoreLog := appstoreLogger()
	appstoreService := appstoresvc.NewService(appstoreLog)
	defer appstoreService.Close()
	appstoreLog.Info("appstore: service ready")

	log.Printf("listening on %s", srv.ClientURL())
	log.Printf("discovery file: %s", bus.DiscoveryPath())

	// Reply handler for on-demand sysinfo requests so a freshly launched
	// TUI doesn't have to wait for the next periodic publish to render.
	if _, err := nc.Subscribe(bus.SubjectSystemInfoCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(sysinfo.Gather(binPath))
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe sysinfo cmd: %v", err)
	}

	// Wifi snapshot responder. We reuse the long-lived Service built at
	// startup so every poll reuses the same D-Bus connection.
	if _, err := nc.Subscribe(bus.SubjectWifiAdaptersCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotWifi(wifiService))
		_ = m.Respond(data)
	}); err != nil {
		log.Fatalf("subscribe wifi adapters cmd: %v", err)
	}

	// Wifi action responders. Each one parses a request, delegates to a
	// handler that talks to the wifi service, then broadcasts the
	// refreshed snapshot so every connected TUI updates in lockstep.
	registerWifiAction := func(subject string, handler func(*wifisvc.Service, wifiActionRequest) wifiActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req wifiActionRequest
			_ = json.Unmarshal(m.Data, &req)
			resp := handler(wifiService, req)
			data, _ := json.Marshal(resp)
			_ = m.Respond(data)
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				_ = nc.Publish(bus.SubjectWifiAdapters, snapData)
			}
		}); err != nil {
			log.Fatalf("subscribe %s: %v", subject, err)
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

	publishBluetooth := wireBluetooth(nc, bluetoothService)
	publishAudio := wireAudio(nc, audioService)
	publishNetwork := wireNetwork(nc, networkService)
	publishAppstore := wireAppstore(nc, appstoreService, appstoreLog)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	heartbeat := time.NewTicker(1 * time.Second)
	defer heartbeat.Stop()

	sysTick := time.NewTicker(2 * time.Second)
	defer sysTick.Stop()

	wifiTick := time.NewTicker(30 * time.Second)
	defer wifiTick.Stop()

	bluetoothTick := time.NewTicker(15 * time.Second)
	defer bluetoothTick.Stop()

	// Audio publishes reactively from the Pulse subscription event
	// stream wired up in wireAudio. This safety-net tick is just for
	// the rare case where we miss an event or reconnect.
	audioTick := time.NewTicker(30 * time.Second)
	defer audioTick.Stop()

	// Network has no event source today (Tier 1 is read-only kernel
	// scrapes); 10 seconds is fast enough that newly-plugged-in
	// ethernet cables and traffic counter rates feel responsive
	// without burning the CPU on sysfs scans.
	networkTick := time.NewTicker(10 * time.Second)
	defer networkTick.Stop()

	// Appstore state barely changes at runtime — the repo list is
	// refreshed by the user via the UI, not by external events. A
	// 60s tick is plenty to pick up installs/removals the user did
	// in another terminal and to keep the installed-set view honest.
	appstoreTick := time.NewTicker(60 * time.Second)
	defer appstoreTick.Stop()

	// Publish initial snapshots immediately so any subscriber that
	// connects in the gap before the first tick still gets pushed data.
	publishSysInfo(nc, binPath)
	publishWifi(nc, wifiService)
	publishBluetooth()
	publishAudio()
	publishNetwork()
	publishAppstore()

	var seq uint64
	for {
		select {
		case <-sigs:
			log.Println("shutting down")
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
			publishSysInfo(nc, binPath)
		case <-wifiTick.C:
			publishWifi(nc, wifiService)
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
	// Give iwd a beat to publish the AP state before we read it back.
	time.Sleep(250 * time.Millisecond)
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
	time.Sleep(250 * time.Millisecond)
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
	time.Sleep(150 * time.Millisecond)
	return wifiActionResponse{Snapshot: svc.Snapshot()}
}

func publishSysInfo(nc *nats.Conn, binPath string) {
	data, err := json.Marshal(sysinfo.Gather(binPath))
	if err != nil {
		log.Printf("marshal sysinfo: %v", err)
		return
	}
	if err := nc.Publish(bus.SubjectSystemInfo, data); err != nil {
		log.Printf("publish sysinfo: %v", err)
	}
}

func publishWifi(nc *nats.Conn, svc *wifisvc.Service) {
	data, err := json.Marshal(snapshotWifi(svc))
	if err != nil {
		log.Printf("marshal wifi: %v", err)
		return
	}
	if err := nc.Publish(bus.SubjectWifiAdapters, data); err != nil {
		log.Printf("publish wifi: %v", err)
	}
}
