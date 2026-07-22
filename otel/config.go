package otel

import "sync/atomic"

// Config 公共 OTel 开关（应用侧使用，不嵌入 initialize DAO）。
//
// 启用规则：
//   - Disabled=true 强制关
//   - Enabled=true 强制开
//   - 皆 false 时跟随 SetupOTelSDK 是否已引导
type Config struct {
	Enabled   bool    `json:"enabled"`
	Disabled  bool    `json:"disabled"`
	SlowSQLMs float64 `json:"slow_sql_ms"`
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
