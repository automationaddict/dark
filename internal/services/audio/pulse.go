package audio

import (
	"fmt"
	"net"
	"os"
	"path"
	"sort"
	"sync"

	"github.com/jfreymuth/pulse/proto"
)

// pulseBackend talks to the host's audio server over the PulseAudio
// native protocol. PipeWire's pipewire-pulse shim implements the same
// protocol, so this backend works transparently on PipeWire systems
// (which is Omarchy's default).
//
// Beyond the request/reply control surface, the backend also runs two
// streaming pipelines off the client's Callback hook:
//
//   - Subscription events (sink/source/card added/removed/changed)
//     are coalesced into a buffered channel the daemon drains so it
//     can republish snapshots reactively instead of polling.
//
//   - Peak-detect record streams are opened against every sink's
//     monitor source and every source's index, producing one float32
//     per chunk that the callback decodes into a per-device level map
//     for the daemon's level meter publisher.
type pulseBackend struct {
	client *proto.Client
	conn   net.Conn
	mu     sync.Mutex

	eventCh chan struct{} // capacity 1, signals "republish snapshot"

	metersMu     sync.Mutex
	meterStreams map[uint32]meterEntry    // record stream index → device key
	sinkLevels   map[uint32][2]float32    // sink index → [left, right] peak
	sourceLevels map[uint32][2]float32    // source index → [left, right] peak
}

// meterEntry maps a record-stream index back to the device kind and
// PulseAudio device index it's measuring. We need this because the
// callback only knows the stream index — it has to translate that to
// "this is the level for sink 5" before updating the levels map.
type meterEntry struct {
	kind        meterKind
	deviceIndex uint32
}

type meterKind int

const (
	meterKindSink meterKind = iota
	meterKindSource
)

func newPulseBackend() (*pulseBackend, error) {
	client, conn, err := proto.Connect("")
	if err != nil {
		return nil, fmt.Errorf("pulse connect: %w", err)
	}
	// Register a client name so our process shows up nicely in
	// pactl list clients / PipeWire's graph view.
	props := proto.PropList{
		"application.name":           proto.PropListString("darkd"),
		"application.process.id":     proto.PropListString(fmt.Sprintf("%d", os.Getpid())),
		"application.process.binary": proto.PropListString(path.Base(os.Args[0])),
	}
	if err := client.Request(&proto.SetClientName{Props: props}, &proto.SetClientNameReply{}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pulse set client name: %w", err)
	}

	b := &pulseBackend{
		client:       client,
		conn:         conn,
		eventCh:      make(chan struct{}, 1),
		meterStreams: map[uint32]meterEntry{},
		sinkLevels:   map[uint32][2]float32{},
		sourceLevels: map[uint32][2]float32{},
	}

	// Install the callback BEFORE subscribing so we don't miss events
	// fired between the Subscribe call and the field assignment.
	client.Callback = b.handleProtoMessage

	if err := client.Request(&proto.Subscribe{Mask: proto.SubscriptionMaskAll}, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pulse subscribe: %w", err)
	}

	return b, nil
}

func (b *pulseBackend) Name() string { return BackendPulseProto }

func (b *pulseBackend) Close() {
	if b.client != nil {
		b.closeAllMeterStreams()
	}
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
	b.client = nil
}

// Snapshot builds the audio payload by issuing several list requests
// against the server. Sorted output keeps rendering stable across
// snapshots — the server returns devices in creation order, which
// changes as hardware plugs and unplugs.
//
// After building the snapshot, Snapshot also reconciles the meter
// streams: for each new sink/source it opens a peak-detect record
// stream, and for any device that has gone away it closes the stale
// stream. The reconcile runs outside b.mu since it issues its own
// proto requests.
func (b *pulseBackend) Snapshot() Snapshot {
	snap := b.snapshotLocked()
	if snap.Backend == BackendPulseProto {
		b.reconcileMeters(snap)
	}
	return snap
}

func (b *pulseBackend) snapshotLocked() Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.client == nil {
		return Snapshot{Backend: BackendNone}
	}

	var server proto.GetServerInfoReply
	if err := b.client.Request(&proto.GetServerInfo{}, &server); err != nil {
		return Snapshot{Backend: BackendPulseProto}
	}

	var sinkReply proto.GetSinkInfoListReply
	_ = b.client.Request(&proto.GetSinkInfoList{}, &sinkReply)

	var sourceReply proto.GetSourceInfoListReply
	_ = b.client.Request(&proto.GetSourceInfoList{}, &sourceReply)

	var cardReply proto.GetCardInfoListReply
	_ = b.client.Request(&proto.GetCardInfoList{}, &cardReply)

	var sinkInputReply proto.GetSinkInputInfoListReply
	_ = b.client.Request(&proto.GetSinkInputInfoList{}, &sinkInputReply)

	var sourceOutputReply proto.GetSourceOutputInfoListReply
	_ = b.client.Request(&proto.GetSourceOutputInfoList{}, &sourceOutputReply)

	snap := Snapshot{
		Backend:       BackendPulseProto,
		DefaultSink:   server.DefaultSinkName,
		DefaultSource: server.DefaultSourceName,
	}

	for _, s := range sinkReply {
		if s == nil {
			continue
		}
		snap.Sinks = append(snap.Sinks, sinkInfoToDevice(s, server.DefaultSinkName))
	}
	for _, s := range sourceReply {
		if s == nil {
			continue
		}
		// PipeWire exposes monitor sources for every sink (virtual
		// "loopback" sources that let you record what's being played).
		// They clutter the UI and aren't something a user meaningfully
		// picks as an input, so filter them out here.
		if isMonitorSource(s) {
			continue
		}
		snap.Sources = append(snap.Sources, sourceInfoToDevice(s, server.DefaultSourceName))
	}
	for _, c := range cardReply {
		if c == nil {
			continue
		}
		snap.Cards = append(snap.Cards, cardInfoToCard(c))
	}

	sortDevices(snap.Sinks)
	sortDevices(snap.Sources)

	// Resolve stream device names against the freshly-built sink and
	// source lists so the TUI can show "Spotify → Built-in Speakers"
	// without doing the lookup itself.
	sinkNameByIndex := map[uint32]string{}
	for _, s := range snap.Sinks {
		sinkNameByIndex[s.Index] = s.Description
		if sinkNameByIndex[s.Index] == "" {
			sinkNameByIndex[s.Index] = s.Name
		}
	}
	sourceNameByIndex := map[uint32]string{}
	for _, s := range snap.Sources {
		sourceNameByIndex[s.Index] = s.Description
		if sourceNameByIndex[s.Index] == "" {
			sourceNameByIndex[s.Index] = s.Name
		}
	}

	for _, si := range sinkInputReply {
		if si == nil {
			continue
		}
		snap.SinkInputs = append(snap.SinkInputs, sinkInputToStream(si, sinkNameByIndex))
	}
	for _, so := range sourceOutputReply {
		if so == nil {
			continue
		}
		// Filter out the Peak Detect / monitor streams that PipeWire's
		// pavucontrol-style clients open continuously. Their MediaName
		// is typically "Peak detect" and they aren't user-facing
		// recording streams. Detecting them precisely requires reading
		// PropList for application.id; for now we just keep everything
		// and let the UI surface them.
		snap.SourceOutputs = append(snap.SourceOutputs, sourceOutputToStream(so, sourceNameByIndex))
	}

	return snap
}

// sortDevices puts the default device first, then orders by description.
func sortDevices(ds []Device) {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i], ds[j]
		if a.IsDefault != b.IsDefault {
			return a.IsDefault
		}
		return a.Description < b.Description
	})
}

// isMonitorSource reports whether a source is PipeWire's auto-generated
// monitor source for a sink. Those have MonitorSourceIndex == Undefined
// on regular sources; actual monitor sources back-reference a sink.
func isMonitorSource(s *proto.GetSourceInfoReply) bool {
	return s.MonitorSourceIndex != proto.Undefined && s.MonitorSourceIndex != ^uint32(0) || s.MonitorSourceName != ""
}

func sinkInfoToDevice(s *proto.GetSinkInfoReply, defaultName string) Device {
	avg := avgVolume(s.ChannelVolumes)
	d := Device{
		Index:        s.SinkIndex,
		Name:         s.SinkName,
		Description:  s.Device,
		CardIndex:    s.CardIndex,
		Mute:         s.Mute,
		Volume:       volumeToPercent(avg),
		VolumeRaw:    avg,
		Channels:     len(s.ChannelVolumes),
		Balance:      computeBalance(s.ChannelVolumes),
		IsDefault:    s.SinkName == defaultName,
		State:        sinkStateString(s.State),
		ActivePort:   s.ActivePortName,
		MonitorIndex: s.MonitorSourceIndex,
	}
	for _, p := range s.Ports {
		d.Ports = append(d.Ports, Port{
			Name:        p.Name,
			Description: p.Description,
			Priority:    p.Priority,
			Available:   p.Available,
		})
	}
	return d
}

func sourceInfoToDevice(s *proto.GetSourceInfoReply, defaultName string) Device {
	avg := avgVolume(s.ChannelVolumes)
	d := Device{
		Index:       s.SourceIndex,
		Name:        s.SourceName,
		Description: s.Device,
		CardIndex:   s.CardIndex,
		Mute:        s.Mute,
		Volume:      volumeToPercent(avg),
		VolumeRaw:   avg,
		Channels:    len(s.ChannelVolumes),
		Balance:     computeBalance(s.ChannelVolumes),
		IsDefault:   s.SourceName == defaultName,
		State:       sinkStateString(s.State),
		ActivePort:  s.ActivePortName,
	}
	for _, p := range s.Ports {
		d.Ports = append(d.Ports, Port{
			Name:        p.Name,
			Description: p.Description,
			Priority:    p.Priority,
			Available:   p.Available,
		})
	}
	return d
}

// cardInfoToCard converts a proto card record to dark's Card type. The
// human-readable description comes from the device.description property
// on the card's PropList — falling back to the raw card name when the
// property is missing.
func cardInfoToCard(c *proto.GetCardInfoReply) Card {
	out := Card{
		Index:         c.CardIndex,
		Name:          c.CardName,
		Driver:        c.Driver,
		ActiveProfile: c.ActiveProfileName,
		Description:   cardDescription(c),
	}
	for _, p := range c.Profiles {
		out.Profiles = append(out.Profiles, Profile{
			Name:        p.Name,
			Description: p.Description,
			NumSinks:    p.NumSinks,
			NumSources:  p.NumSources,
			Priority:    p.Priority,
			Available:   p.Available,
		})
	}
	return out
}

// cardDescription pulls a human-readable label out of a card's
// PropList. PulseAudio cards expose their friendly name as
// "device.description"; the raw CardName is something like
// "alsa_card.pci-0000_00_1f.3" which is too verbose for a UI.
func cardDescription(c *proto.GetCardInfoReply) string {
	for _, key := range []string{"device.description", "alsa.card_name", "bluez.alias"} {
		if v, ok := c.Properties[key]; ok {
			if s := v.String(); s != "" && s != "<not a string>" {
				return s
			}
		}
	}
	return c.CardName
}

// avgVolume is the per-channel average of a ChannelVolumes vector.
// Mirrors libpulse's pa_cvolume_avg.
func avgVolume(cv proto.ChannelVolumes) uint32 {
	if len(cv) == 0 {
		return 0
	}
	var sum uint64
	for _, v := range cv {
		sum += uint64(v)
	}
	return uint32(sum / uint64(len(cv)))
}

// volumeToPercent converts a raw PulseAudio volume to a 0-100+ integer
// percentage using proto.VolumeNorm as the 100% reference.
func volumeToPercent(raw uint32) int {
	return int(uint64(raw) * 100 / uint64(proto.VolumeNorm))
}

// percentToVolumes builds a ChannelVolumes slice with the given
// percentage applied uniformly across `channels` channels. Clamps
// to the [0, 150] range PulseAudio considers "safe" — values above
// 100% enter software over-amplification territory which can clip.
func percentToVolumes(pct int, channels int) proto.ChannelVolumes {
	if channels <= 0 {
		channels = 2
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 150 {
		pct = 150
	}
	raw := uint32(uint64(pct) * uint64(proto.VolumeNorm) / 100)
	out := make(proto.ChannelVolumes, channels)
	for i := range out {
		out[i] = raw
	}
	return out
}

// computeBalance derives a -100..+100 balance from a stereo ChannelVolumes.
// -100 = full left, 0 = center, +100 = full right. Mono returns 0.
func computeBalance(cv proto.ChannelVolumes) int {
	if len(cv) < 2 {
		return 0
	}
	left := float64(cv[0])
	right := float64(cv[1])
	sum := left + right
	if sum == 0 {
		return 0
	}
	// balance = (right - left) / max(left, right) * 100
	max := left
	if right > max {
		max = right
	}
	return int((right - left) / max * 100)
}

// balanceToVolumes builds a ChannelVolumes with the given overall volume
// percentage and balance (-100..+100). The louder channel gets the full
// volume; the quieter channel is scaled down proportionally.
func balanceToVolumes(pct, balance, channels int) proto.ChannelVolumes {
	if channels <= 0 {
		channels = 2
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 150 {
		pct = 150
	}
	raw := uint32(uint64(pct) * uint64(proto.VolumeNorm) / 100)
	out := make(proto.ChannelVolumes, channels)

	if channels < 2 || balance == 0 {
		for i := range out {
			out[i] = raw
		}
		return out
	}

	var leftScale, rightScale float64
	if balance < 0 {
		leftScale = 1.0
		rightScale = 1.0 + float64(balance)/100.0
	} else {
		leftScale = 1.0 - float64(balance)/100.0
		rightScale = 1.0
	}

	out[0] = uint32(float64(raw) * leftScale)
	out[1] = uint32(float64(raw) * rightScale)
	for i := 2; i < channels; i++ {
		out[i] = raw
	}
	return out
}

// sinkStateString maps the raw state enum PulseAudio returns to a
// short label. Values: 0=running, 1=idle, 2=suspended, 3=invalid,
// 4=init, 5=unlinked.
func sinkStateString(state uint32) string {
	switch state {
	case 0:
		return "running"
	case 1:
		return "idle"
	case 2:
		return "suspended"
	case 3:
		return "invalid"
	case 4:
		return "init"
	case 5:
		return "unlinked"
	default:
		return ""
	}
}

// --- actions ---

func (b *pulseBackend) SetSinkVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sinkChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSinkVolume{
		SinkIndex:      index,
		ChannelVolumes: percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkBalance(index uint32, balance int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	reply := &proto.GetSinkInfoReply{}
	if err := b.client.Request(&proto.GetSinkInfo{SinkIndex: index}, reply); err != nil {
		return fmt.Errorf("get sink info: %w", err)
	}
	pct := volumeToPercent(avgVolume(reply.ChannelVolumes))
	channels := len(reply.ChannelVolumes)
	req := &proto.SetSinkVolume{
		SinkIndex:      index,
		ChannelVolumes: balanceToVolumes(pct, balance, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink balance: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSinkMute{SinkIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sourceChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSourceVolume{
		SourceIndex:    index,
		ChannelVolumes: percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSourceMute{SourceIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceBalance(index uint32, balance int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	reply := &proto.GetSourceInfoReply{}
	if err := b.client.Request(&proto.GetSourceInfo{SourceIndex: index}, reply); err != nil {
		return fmt.Errorf("get source info: %w", err)
	}
	pct := volumeToPercent(avgVolume(reply.ChannelVolumes))
	channels := len(reply.ChannelVolumes)
	req := &proto.SetSourceVolume{
		SourceIndex:    index,
		ChannelVolumes: balanceToVolumes(pct, balance, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source balance: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetDefaultSink(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.SetDefaultSink{SinkName: name}, nil); err != nil {
		return fmt.Errorf("set default sink: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetDefaultSource(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.SetDefaultSource{SourceName: name}, nil); err != nil {
		return fmt.Errorf("set default source: %w", err)
	}
	return nil
}

// sinkInputToStream converts a proto sink input record to dark's
// Stream type. The application name comes from the PropList's
// application.name property; we fall back to the proto MediaName for
// streams that don't set it.
func sinkInputToStream(si *proto.GetSinkInputInfoReply, sinkNames map[uint32]string) Stream {
	avg := avgVolume(si.ChannelVolumes)
	return Stream{
		Index:       si.SinkInputIndex,
		DeviceIndex: si.SinkIndex,
		DeviceName:  sinkNames[si.SinkIndex],
		Application: streamApplicationName(si.Properties),
		MediaName:   si.MediaName,
		Mute:        si.Muted,
		Volume:      volumeToPercent(avg),
		VolumeRaw:   avg,
		Channels:    len(si.ChannelVolumes),
		Corked:      si.Corked,
	}
}

func sourceOutputToStream(so *proto.GetSourceOutputInfoReply, sourceNames map[uint32]string) Stream {
	avg := avgVolume(so.ChannelVolumes)
	return Stream{
		Index:       so.SourceOutpuIndex,
		DeviceIndex: so.SourceIndex,
		DeviceName:  sourceNames[so.SourceIndex],
		Application: streamApplicationName(so.Properties),
		MediaName:   so.MediaName,
		Mute:        so.Muted,
		Volume:      volumeToPercent(avg),
		VolumeRaw:   avg,
		Channels:    len(so.ChannelVolumes),
		Corked:      so.Corked,
	}
}

// streamApplicationName extracts the user-facing app name from a
// stream's PropList. Tries application.name first, then process.binary,
// then media.role as a last resort.
func streamApplicationName(props proto.PropList) string {
	for _, key := range []string{"application.name", "application.process.binary", "media.role"} {
		if v, ok := props[key]; ok {
			s := v.String()
			if s != "" && s != "<not a string>" {
				return s
			}
		}
	}
	return ""
}

func (b *pulseBackend) SetCardProfile(cardIndex uint32, profile string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetCardProfile{CardIndex: cardIndex, ProfileName: profile}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set card profile: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkPort(sinkIndex uint32, port string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSinkPort{SinkIndex: sinkIndex, Port: port}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink port: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourcePort(sourceIndex uint32, port string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSourcePort{SourceIndex: sourceIndex, Port: port}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source port: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkInputVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sinkInputChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSinkInputVolume{
		SinkInputIndex: index,
		ChannelVolumes: percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink input volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSinkInputMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSinkInputMute{SinkInputIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set sink input mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) MoveSinkInput(streamIndex, sinkIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.MoveSinkInput{SinkInputIndex: streamIndex, DeviceIndex: sinkIndex}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("move sink input: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceOutputVolume(index uint32, pct int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	channels, err := b.sourceOutputChannelCount(index)
	if err != nil {
		return err
	}
	req := &proto.SetSourceOutputVolume{
		SourceOutputIndex: index,
		ChannelVolumes:    percentToVolumes(pct, channels),
	}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source output volume: %w", err)
	}
	return nil
}

func (b *pulseBackend) SetSourceOutputMute(index uint32, mute bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SetSourceOutputMute{SourceOutputIndex: index, Mute: mute}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("set source output mute: %w", err)
	}
	return nil
}

func (b *pulseBackend) MoveSourceOutput(streamIndex, sourceIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.MoveSourceOutput{SourceOutputIndex: streamIndex, DeviceIndex: sourceIndex}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("move source output: %w", err)
	}
	return nil
}

func (b *pulseBackend) KillSinkInput(streamIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.KillSinkInput{SinkInputIndex: streamIndex}, nil); err != nil {
		return fmt.Errorf("kill sink input: %w", err)
	}
	return nil
}

func (b *pulseBackend) KillSourceOutput(streamIndex uint32) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.client.Request(&proto.KillSourceOutput{SourceOutputIndex: streamIndex}, nil); err != nil {
		return fmt.Errorf("kill source output: %w", err)
	}
	return nil
}

func (b *pulseBackend) SuspendSink(index uint32, suspend bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SuspendSink{SinkIndex: index, Suspend: suspend}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("suspend sink: %w", err)
	}
	return nil
}

func (b *pulseBackend) SuspendSource(index uint32, suspend bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	req := &proto.SuspendSource{SourceIndex: index, Suspend: suspend}
	if err := b.client.Request(req, nil); err != nil {
		return fmt.Errorf("suspend source: %w", err)
	}
	return nil
}

// sinkInputChannelCount re-reads the live channel count for a sink
// input so the ChannelVolumes vector we send matches what the server
// expects.
func (b *pulseBackend) sinkInputChannelCount(index uint32) (int, error) {
	var reply proto.GetSinkInputInfoReply
	if err := b.client.Request(&proto.GetSinkInputInfo{SinkInputIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get sink input info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}

func (b *pulseBackend) sourceOutputChannelCount(index uint32) (int, error) {
	var reply proto.GetSourceOutputInfoReply
	if err := b.client.Request(&proto.GetSourceOutputInfo{SourceOutpuIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get source output info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}

// sinkChannelCount re-reads the live channel count for a sink so the
// ChannelVolumes vector we send matches what the server expects. The
// snapshot cached in the TUI may be stale by a few ticks.
func (b *pulseBackend) sinkChannelCount(index uint32) (int, error) {
	var reply proto.GetSinkInfoReply
	if err := b.client.Request(&proto.GetSinkInfo{SinkIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get sink info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}

func (b *pulseBackend) sourceChannelCount(index uint32) (int, error) {
	var reply proto.GetSourceInfoReply
	if err := b.client.Request(&proto.GetSourceInfo{SourceIndex: index}, &reply); err != nil {
		return 0, fmt.Errorf("get source info: %w", err)
	}
	return len(reply.ChannelVolumes), nil
}
