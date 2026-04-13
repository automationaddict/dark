package audio

import (
	"sort"
	"strings"

	"github.com/jfreymuth/pulse/proto"
)

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
		if isMonitorStream(so.Properties, so.MediaName) {
			continue
		}
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

// isMonitorStream returns true for PipeWire/PulseAudio internal streams
// that aren't user-facing recording applications. These include peak-detect
// streams (used for VU meters), PipeWire Manager housekeeping, and any
// stream whose media.role is explicitly "abstract" or "filter".
func isMonitorStream(props proto.PropList, mediaName string) bool {
	if strings.EqualFold(mediaName, "Peak detect") {
		return true
	}
	for _, key := range []string{"application.id", "application.name", "application.process.binary"} {
		if v, ok := props[key]; ok {
			s := v.String()
			switch {
			case strings.Contains(s, "peak detect"):
				return true
			case s == "darkd":
				return true
			case s == "PipeWire Manager":
				return true
			case s == "pipewire-media-session":
				return true
			case s == "wireplumber":
				return true
			}
		}
	}
	if v, ok := props["media.role"]; ok {
		role := v.String()
		if role == "abstract" || role == "filter" {
			return true
		}
	}
	return false
}
