package otel

import "sync/atomic"

// Config is a common on/off switch that I/O plugins can embed or configure independently.
// Disabled=true forces the plugin off; Enabled=true forces it on; both false follows SetupOTelSDK.
type Config struct {
	Enabled  bool `json:"enabled"`
	Disabled bool `json:"disabled"`
}

// Active reports whether this plugin should be enabled given the current bootstrapping state.
func (c Config) Active() bool {
	if c.Disabled {
		return false
	}
	if c.Enabled {
		return true
	}
	return IsBootstrapped()
}

var bootstrapped atomic.Bool

func IsBootstrapped() bool { return bootstrapped.Load() }

// markBootstrapped records that the OTel SDK has been fully initialized.
func markBootstrapped() { bootstrapped.Store(true) }
