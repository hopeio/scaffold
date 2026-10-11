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
// A missing client is a no-op. Callers that hold a partially assembled Dao (for
// example one built with RootConfig.SkipInjectDaos) pass a nil *redis.Client,
// which lands here as a non-nil interface carrying a nil pointer, so the check
// must be the typed-nil aware isNilValue rather than a plain == nil.
func (p *RedisPlugin) Instrument(client redis.UniversalClient) error {
	if p == nil || isNilValue(client) {
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

// isNilValue reports whether v is untyped nil or an interface carrying a nil
// pointer, map, slice, func, chan or unsafe pointer. Passing (*redis.Client)(nil)
// into an interface parameter yields a non-nil interface, which == nil misses.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return rv.IsNil()
	default:
		return false
	}
}
