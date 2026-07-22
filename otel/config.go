package otel

import "sync/atomic"

// Config 公共 OTel 开关，可嵌入各 DAO Config。
//
// 启用规则：
//   - Disabled=true 时强制关闭
//   - Enabled=true 时强制开启
//   - 二者皆为 false 时，跟随 SetupOTelSDK 是否已成功引导
type Config struct {
	Enabled   bool    `json:"enabled"`
	Disabled  bool    `json:"disabled"`
	SlowSQLMs float64 `json:"slow_sql_ms"` // GORM 慢 SQL 阈值（毫秒），0 用默认 200
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

// IsBootstrapped 报告 SetupOTelSDK 是否已成功执行。
func IsBootstrapped() bool {
	return bootstrapped.Load()
}

func markBootstrapped() {
	bootstrapped.Store(true)
}
