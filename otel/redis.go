package otel

import (
	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// RedisPlugin attaches OTel tracing and/or metrics to a go-redis client via hooks.
type RedisPlugin struct {
	Config
	Tracing bool // when both Tracing and Metrics are false, both default to true
	Metrics bool
}

// NewRedisPlugin creates a RedisPlugin; enables both tracing and metrics when neither is explicitly set.
func NewRedisPlugin(cfg RedisPlugin) *RedisPlugin {
	if !cfg.Tracing && !cfg.Metrics {
		cfg.Tracing, cfg.Metrics = true, true
	}
	return &cfg
}

// Instrument installs redisotel tracing and/or metrics hooks on the given client.
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
