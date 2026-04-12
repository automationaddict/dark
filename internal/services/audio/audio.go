// Package audio talks to the host's audio stack via the PulseAudio
// native protocol (which PipeWire's pipewire-pulse shim implements
// transparently). All stack-specific logic lives behind the Backend
// interface so a future native PipeWire client could slot in without
// touching the TUI or the daemon.
package audio

// Device is one sink (output) or source (input) from the server. Sinks
// and sources share the same shape since PulseAudio's protocol defines
// them with near-identical fields. Context (output vs input) comes from
// the list they live in on the Snapshot.
type Device struct {
	Index       uint32 `json:"index"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CardIndex   uint32 `json:"card_index,omitempty"` // PulseAudio "undefined" when not card-backed
	Mute        bool   `json:"mute"`
	Volume      int    `json:"volume"`   // 0 - 100+ (over-amp goes above 100)
	VolumeRaw   uint32 `json:"volume_raw"` // raw PulseAudio Volume for round-tripping
	Channels    int    `json:"channels"` // number of channels so SetVolume can build a correct ChannelVolumes
	Balance     int    `json:"balance"`  // -100 (full left) to +100 (full right), 0 = center
	IsDefault   bool   `json:"is_default"`
	State       string `json:"state,omitempty"`

	// Ports carry the list of physical jacks/routes exposed by the
	// backing card. Typically "headphones" and "speaker" on a laptop
	// internal card, or empty on sinks with no routing concept (like
	// a virtual pipewire-pulse null sink).
	Ports      []Port `json:"ports,omitempty"`
	ActivePort string `json:"active_port,omitempty"`

	// MonitorIndex is the PulseAudio source index of this sink's
	// monitor source — the virtual source that lets you record
	// whatever the sink is currently playing. Used by the level
	// metering pipeline to attach a peak-detect stream. Sources
	// don't have a monitor; this field is meaningful only for sinks.
	MonitorIndex uint32 `json:"monitor_index,omitempty"`
}

// Port is one routable jack/output exposed by a card. Direction is
// derived from whether the parent device is a sink or a source — we
// don't carry it on the port itself.
type Port struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Priority    uint32 `json:"priority,omitempty"`
	// Available: 0 = unknown, 1 = not available (e.g. headphones jack
	// with nothing plugged in), 2 = available.
	Available uint32 `json:"available,omitempty"`
}

// Card is one hardware device the audio server has opened. Cards own
// sinks, sources, and profiles. Switching a profile reconfigures the
// sinks/sources the card exposes — this is where bluetooth A2DP vs
// HFP switching happens, and where laptop "analog stereo" vs "HDMI"
// selection lives on systems with multiple integrated outputs.
type Card struct {
	Index         uint32    `json:"index"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	Driver        string    `json:"driver,omitempty"`
	Profiles      []Profile `json:"profiles,omitempty"`
	ActiveProfile string    `json:"active_profile,omitempty"`
}

// Profile is one operating mode a card can be switched into. For a
// bluetooth headset this looks like `a2dp-sink`, `headset-head-unit`,
// `off`. For an analog laptop card it's `output:analog-stereo`,
// `input:analog-stereo`, `off`.
type Profile struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	NumSinks    uint32 `json:"num_sinks,omitempty"`
	NumSources  uint32 `json:"num_sources,omitempty"`
	Priority    uint32 `json:"priority,omitempty"`
	// Available: 0 = unknown, 1 = not available, 2 = available. Some
	// profiles (like bluetooth HFP) only become available when the
	// remote end negotiates support, so we show them greyed out when
	// the server reports them as unavailable.
	Available uint32 `json:"available,omitempty"`
}

// Stream is one per-application audio stream — either a sink input
// (an app playing into a sink) or a source output (an app recording
// from a source). The two PulseAudio types share enough fields that
// dark uses a single struct distinguished only by the list it lives
// in on the Snapshot.
type Stream struct {
	Index       uint32 `json:"index"`
	DeviceIndex uint32 `json:"device_index"` // sink index for sink inputs, source index for source outputs
	DeviceName  string `json:"device_name,omitempty"`
	Application string `json:"application,omitempty"` // application.name from PropList
	MediaName   string `json:"media_name,omitempty"`  // descriptive media stream name
	Mute        bool   `json:"mute"`
	Volume      int    `json:"volume"`
	VolumeRaw   uint32 `json:"volume_raw"`
	Channels    int    `json:"channels"`
	Corked      bool   `json:"corked,omitempty"`
}

// DisplayName returns the most user-friendly label for this stream:
// the app name when set, falling back to the media name, then to a
// generic placeholder. PulseAudio's MediaName for a music app is
// usually the track title, which is more useful than the app name.
func (s Stream) DisplayName() string {
	switch {
	case s.Application != "" && s.MediaName != "":
		return s.Application + " · " + s.MediaName
	case s.Application != "":
		return s.Application
	case s.MediaName != "":
		return s.MediaName
	default:
		return "(unnamed stream)"
	}
}

// Snapshot is the audio domain payload published on the bus.
type Snapshot struct {
	Backend       string   `json:"backend"`
	Sinks         []Device `json:"sinks"`
	Sources       []Device `json:"sources"`
	Cards         []Card   `json:"cards,omitempty"`
	SinkInputs    []Stream `json:"sink_inputs,omitempty"`
	SourceOutputs []Stream `json:"source_outputs,omitempty"`
	DefaultSink   string   `json:"default_sink,omitempty"`
	DefaultSource string   `json:"default_source,omitempty"`
}

// Levels is a snapshot of the most recent peak readings for every
// sink and source the daemon has a meter stream open against. Each
// value is a stereo pair [left, right] normalized to [0.0, 1.0] —
// PulseAudio's peak-detect mode emits one float per channel per
// chunk. Mono sources are upmixed by the server when we ask for two
// channels, so both halves are equal for a mono mic.
type Levels struct {
	Sinks   map[uint32][2]float32 `json:"sinks,omitempty"`
	Sources map[uint32][2]float32 `json:"sources,omitempty"`
}

// CardByIndex returns the card with the given index, if any. The bool
// is false when the index is PulseAudio's "undefined" sentinel or
// simply not present (e.g. virtual null sinks aren't card-backed).
func (s Snapshot) CardByIndex(index uint32) (Card, bool) {
	for _, c := range s.Cards {
		if c.Index == index {
			return c, true
		}
	}
	return Card{}, false
}

// Backend identifiers.
const (
	BackendNone       = "none"
	BackendPulseProto = "pulse"
)

// Service owns the chosen Backend and is the single entry point the
// daemon uses to read or mutate audio state.
type Service struct {
	backend Backend
}

// NewService opens a connection to the audio server and returns a
// Service wired to the right backend. Falls back to a noop backend
// when the server can't be reached.
func NewService() (*Service, error) {
	backend, err := newPulseBackend()
	if err != nil {
		return &Service{backend: newNoopBackend()}, err
	}
	return &Service{backend: backend}, nil
}

func (s *Service) Close() {
	if s.backend != nil {
		s.backend.Close()
		s.backend = nil
	}
}

func (s *Service) Snapshot() Snapshot {
	if s.backend == nil {
		return Snapshot{Backend: BackendNone}
	}
	return s.backend.Snapshot()
}

// Events returns a channel that fires whenever the audio server
// reports a property change (sink/source/card added, removed, or
// modified). The daemon listens on this channel and republishes the
// audio snapshot reactively, replacing the polling tick. The channel
// is buffered and the backend never blocks on send, so consumers may
// drop events under load — that's fine because the next event will
// trigger another snapshot anyway.
func (s *Service) Events() <-chan struct{} {
	if s.backend == nil {
		return nil
	}
	return s.backend.Events()
}

// Levels returns the current peak meter readings. Safe to call from
// any goroutine. Returns a copy so callers can hold it without
// worrying about racing the backend.
func (s *Service) Levels() Levels {
	if s.backend == nil {
		return Levels{}
	}
	return s.backend.Levels()
}

// Detect is a one-shot convenience. Used by the daemon snapshot reply
// if the long-lived Service couldn't be built.
func Detect() Snapshot {
	svc, err := NewService()
	if err != nil {
		return Snapshot{Backend: BackendNone}
	}
	defer svc.Close()
	return svc.Snapshot()
}

// --- action delegation ---

func (s *Service) SetSinkVolume(index uint32, pct int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSinkVolume(index, pct)
}

func (s *Service) SetSinkMute(index uint32, mute bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSinkMute(index, mute)
}

func (s *Service) SetSinkBalance(index uint32, balance int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSinkBalance(index, balance)
}

func (s *Service) SetSourceVolume(index uint32, pct int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSourceVolume(index, pct)
}

func (s *Service) SetSourceMute(index uint32, mute bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSourceMute(index, mute)
}

func (s *Service) SetSourceBalance(index uint32, balance int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSourceBalance(index, balance)
}

func (s *Service) SetDefaultSink(name string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetDefaultSink(name)
}

func (s *Service) SetDefaultSource(name string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetDefaultSource(name)
}

func (s *Service) SetCardProfile(cardIndex uint32, profile string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetCardProfile(cardIndex, profile)
}

func (s *Service) SetSinkPort(sinkIndex uint32, port string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSinkPort(sinkIndex, port)
}

func (s *Service) SetSourcePort(sourceIndex uint32, port string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSourcePort(sourceIndex, port)
}

func (s *Service) SetSinkInputVolume(index uint32, pct int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSinkInputVolume(index, pct)
}

func (s *Service) SetSinkInputMute(index uint32, mute bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSinkInputMute(index, mute)
}

func (s *Service) MoveSinkInput(streamIndex, sinkIndex uint32) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.MoveSinkInput(streamIndex, sinkIndex)
}

func (s *Service) SetSourceOutputVolume(index uint32, pct int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSourceOutputVolume(index, pct)
}

func (s *Service) SetSourceOutputMute(index uint32, mute bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSourceOutputMute(index, mute)
}

func (s *Service) MoveSourceOutput(streamIndex, sourceIndex uint32) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.MoveSourceOutput(streamIndex, sourceIndex)
}

func (s *Service) KillSinkInput(streamIndex uint32) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.KillSinkInput(streamIndex)
}

func (s *Service) KillSourceOutput(streamIndex uint32) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.KillSourceOutput(streamIndex)
}

func (s *Service) SuspendSink(index uint32, suspend bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SuspendSink(index, suspend)
}

func (s *Service) SuspendSource(index uint32, suspend bool) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SuspendSource(index, suspend)
}
