package otel

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

// A skipped Dao leaves its client fields as nil pointers. Those reach Instrument
// through the redis.UniversalClient interface as typed nils, which a plain == nil
// check does not catch and redisotel then dereferences.
func TestRedisPlugin_InstrumentNilClients(t *testing.T) {
	tests := []struct {
		name   string
		client redis.UniversalClient
	}{
		{"untyped nil", nil},
		{"nil *redis.Client", (*redis.Client)(nil)},
		{"nil *redis.ClusterClient", (*redis.ClusterClient)(nil)},
		{"nil *redis.Ring", (*redis.Ring)(nil)},
	}
	p := NewRedisPlugin(RedisPlugin{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := p.Instrument(tt.client); err != nil {
				t.Fatalf("Instrument() = %v, want nil", err)
			}
		})
	}
}

// A nil plugin must stay a no-op even for a usable client.
func TestRedisPlugin_InstrumentNilPlugin(t *testing.T) {
	var p *RedisPlugin
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = client.Close() }()
	if err := p.Instrument(client); err != nil {
		t.Fatalf("Instrument() = %v, want nil", err)
	}
}
