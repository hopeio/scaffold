package otel

import "sync/atomic"

// Config 通用开关；各 I/O 插件可内嵌或单独配置。
//
//	Disabled=true 强制关；Enabled=true 强制开；皆 false 则跟随 SetupOTelSDK。
type Config struct {
	Enabled  bool `json:"enabled"`
	Disabled bool `json:"disabled"`
}

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

func markBootstrapped() { bootstrapped.Store(true) }
