package otel

import (
	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// RedisPlugin 通过 go-redis Hook 挂载 OTel。
type RedisPlugin struct {
	Config
	Tracing bool // 与 Metrics 皆为 false 时默认都开
	Metrics bool
}

func NewRedisPlugin(cfg RedisPlugin) *RedisPlugin {
	if !cfg.Tracing && !cfg.Metrics {
		cfg.Tracing, cfg.Metrics = true, true
	}
	return &cfg
}

// Instrument 对已创建的 client 调用 InstrumentTracing / InstrumentMetrics。
func (p *RedisPlugin) Instrument(client redis.UniversalClient) error {
	if p == nil || !p.Active() || client == nil {
		return nil
	}
	if p.Tracing {
		if err := redisotel.InstrumentTracing(client); err != nil {
			return err
		}
	}
	if p.Metrics {
		if err := redisotel.InstrumentMetrics(client); err != nil {
			return err
		}
	}
	return nil
}
