package otel

import (
	"reflect"

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
// A nil client is legal: SkipInjectDaos (test/CI config) leaves the field as a
// typed-nil pointer, which a plain `client == nil` check misses.
func (p *RedisPlugin) Instrument(client redis.UniversalClient) error {
	if p == nil || isNilClient(client) {
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

// isNilClient reports whether the interface holds a typed-nil pointer — a plain
// `client == nil` comparison misses a nil *redis.Client stored inside the interface.
func isNilClient(client redis.UniversalClient) bool {
	if client == nil {
		return true
	}
	v := reflect.ValueOf(client)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
