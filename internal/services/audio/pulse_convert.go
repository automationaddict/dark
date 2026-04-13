package audio

import "github.com/jfreymuth/pulse/proto"

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
		MonitorName:  s.MonitorSourceName,
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
