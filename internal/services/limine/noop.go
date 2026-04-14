package limine

// noopBackend stands in when limine isn't installed on the host. The
// daemon still runs and the F3 limine view simply shows "unavailable".
type noopBackend struct{}

func newNoopBackend() *noopBackend { return &noopBackend{} }

func (n *noopBackend) Name() string                           { return BackendNone }
func (n *noopBackend) Snapshot() Snapshot                     { return Snapshot{Backend: BackendNone} }
func (n *noopBackend) CreateSnapshot(string) error            { return ErrBackendUnsupported }
func (n *noopBackend) DeleteSnapshot(int) error               { return ErrBackendUnsupported }
func (n *noopBackend) Sync() error                            { return ErrBackendUnsupported }
func (n *noopBackend) SetDefaultEntry(int) error              { return ErrBackendUnsupported }
func (n *noopBackend) SetBootConfig(string, string) error     { return ErrBackendUnsupported }
func (n *noopBackend) SetSyncConfig(string, string) error     { return ErrBackendUnsupported }
func (n *noopBackend) SetOmarchyConfig(string, string) error  { return ErrBackendUnsupported }
func (n *noopBackend) SetOmarchyKernelCmdline([]string) error { return ErrBackendUnsupported }
func (n *noopBackend) Close()                                 {}
