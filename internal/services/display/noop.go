package display

type noopBackend struct{}

func newNoopBackend() *noopBackend { return &noopBackend{} }

func (b *noopBackend) Name() string       { return "none" }
func (b *noopBackend) Snapshot() Snapshot  { return Snapshot{} }
func (b *noopBackend) Close()              {}
func (b *noopBackend) Events() <-chan struct{} { return nil }

func (b *noopBackend) SetResolution(string, int, int, float64) error { return ErrBackendUnavailable }
func (b *noopBackend) SetScale(string, float64) error                { return ErrBackendUnavailable }
func (b *noopBackend) SetTransform(string, int) error                { return ErrBackendUnavailable }
func (b *noopBackend) SetPosition(string, int, int) error            { return ErrBackendUnavailable }
func (b *noopBackend) SetDpms(string, bool) error                    { return ErrBackendUnavailable }
func (b *noopBackend) SetVrr(string, int) error                      { return ErrBackendUnavailable }
func (b *noopBackend) SetMirror(string, string) error                { return ErrBackendUnavailable }
func (b *noopBackend) ToggleEnabled(string) error                    { return ErrBackendUnavailable }
func (b *noopBackend) Identify() error                               { return ErrBackendUnavailable }
func (b *noopBackend) SetBrightness(int) error                       { return ErrBackendUnavailable }
func (b *noopBackend) SetKbdBrightness(int) error                    { return ErrBackendUnavailable }
func (b *noopBackend) SetNightLight(bool, int, int) error             { return ErrBackendUnavailable }
func (b *noopBackend) SetGamma(int) error                             { return ErrBackendUnavailable }
func (b *noopBackend) SaveProfile(string) error                       { return ErrBackendUnavailable }
func (b *noopBackend) ApplyProfile(string) error                      { return ErrBackendUnavailable }
func (b *noopBackend) DeleteProfile(string) error                     { return ErrBackendUnavailable }
