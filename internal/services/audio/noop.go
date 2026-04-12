package audio

// noopBackend stands in when no audio server is reachable. Snapshot
// returns an empty payload and every action returns ErrBackendUnsupported.
type noopBackend struct{}

func newNoopBackend() *noopBackend { return &noopBackend{} }

func (n *noopBackend) Name() string                       { return BackendNone }
func (n *noopBackend) Snapshot() Snapshot                 { return Snapshot{Backend: BackendNone} }
func (n *noopBackend) Events() <-chan struct{}            { return nil }
func (n *noopBackend) Levels() Levels                     { return Levels{} }
func (n *noopBackend) Close()                             {}
func (n *noopBackend) SetSinkVolume(uint32, int) error      { return ErrBackendUnsupported }
func (n *noopBackend) SetSinkMute(uint32, bool) error       { return ErrBackendUnsupported }
func (n *noopBackend) SetSourceVolume(uint32, int) error    { return ErrBackendUnsupported }
func (n *noopBackend) SetSourceMute(uint32, bool) error     { return ErrBackendUnsupported }
func (n *noopBackend) SetDefaultSink(string) error          { return ErrBackendUnsupported }
func (n *noopBackend) SetDefaultSource(string) error        { return ErrBackendUnsupported }
func (n *noopBackend) SetCardProfile(uint32, string) error      { return ErrBackendUnsupported }
func (n *noopBackend) SetSinkPort(uint32, string) error         { return ErrBackendUnsupported }
func (n *noopBackend) SetSourcePort(uint32, string) error       { return ErrBackendUnsupported }
func (n *noopBackend) SetSinkInputVolume(uint32, int) error     { return ErrBackendUnsupported }
func (n *noopBackend) SetSinkInputMute(uint32, bool) error      { return ErrBackendUnsupported }
func (n *noopBackend) MoveSinkInput(uint32, uint32) error       { return ErrBackendUnsupported }
func (n *noopBackend) KillSinkInput(uint32) error               { return ErrBackendUnsupported }
func (n *noopBackend) SetSourceOutputVolume(uint32, int) error  { return ErrBackendUnsupported }
func (n *noopBackend) SetSourceOutputMute(uint32, bool) error   { return ErrBackendUnsupported }
func (n *noopBackend) MoveSourceOutput(uint32, uint32) error    { return ErrBackendUnsupported }
func (n *noopBackend) KillSourceOutput(uint32) error            { return ErrBackendUnsupported }
func (n *noopBackend) SuspendSink(uint32, bool) error           { return ErrBackendUnsupported }
func (n *noopBackend) SuspendSource(uint32, bool) error         { return ErrBackendUnsupported }
