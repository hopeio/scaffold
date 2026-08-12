package otel

import (
	"sync/atomic"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

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

// SDKConfig configures OpenTelemetry SDK bootstrap (trace sampling, etc.).
type SDKConfig struct {
	// SampleRatio is the fraction of traces to record, clamped to [0, 1].
	// 1 = always sample, 0 = never sample. Default when unset in callers is 1.
	SampleRatio float64
	// Pyroscope：仅 Enabled=true 时启动；地址可从 ServerAddress 或 PYROSCOPE_SERVER_ADDRESS 补。
	Pyroscope PyroscopeConfig
}

// Sampler returns a ParentBased sampler using SampleRatio for both roots and
// remote parents that were not sampled. Without the latter, gateways that inject
// traceparent with sampled=0 would suppress every span.
func (c SDKConfig) Sampler() sdktrace.Sampler {
	root := ratioSampler(c.SampleRatio)
	return sdktrace.ParentBased(root,
		sdktrace.WithRemoteParentNotSampled(root),
	)
}

func ratioSampler(ratio float64) sdktrace.Sampler {
	switch {
	case ratio >= 1:
		return sdktrace.AlwaysSample()
	case ratio <= 0:
		return sdktrace.NeverSample()
	default:
		return sdktrace.TraceIDRatioBased(ratio)
	}
}

var bootstrapped atomic.Bool

func IsBootstrapped() bool { return bootstrapped.Load() }

// markBootstrapped records that the OTel SDK has been fully initialized.
func markBootstrapped() { bootstrapped.Store(true) }
