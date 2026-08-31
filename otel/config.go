package otel

import (
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config is the common embedding point for I/O plugins. Plugins are mounted by the
// application at its own injection hooks (e.g. dao.AfterInject); NOT mounting a plugin
// is how it is disabled — there is no separate on/off flag. This avoids a bootstrap-order
// dependency: a flag whose "auto" state depends on whether SetupOTelSDK has run would
// silently no-op at injection time (before the SDK exists), dropping DB/Redis/S3 spans.
type Config struct{}

// SDKConfig configures OpenTelemetry SDK bootstrap (trace sampling, etc.).
type SDKConfig struct {
	// SampleRatio is the fraction of traces to record, clamped to [0, 1].
	// 1 = always sample, 0 = never sample. Default when unset in callers is 1.
	SampleRatio float64
	// Secure 为 true 时不再强制 WithInsecure，交给 OTEL_EXPORTER_OTLP_* 环境变量
	// 与 TLS 默认逻辑；false 保持既有行为（明文导出，适合本地 collector）。
	Secure bool
	// MetricInterval 为周期性指标导出间隔，0 时默认 10s。
	MetricInterval time.Duration
	// DisableRuntimeMetrics 跳过 Go runtime 指标采集。
	DisableRuntimeMetrics bool
	// Export OTLP endpoints/protocol/headers; empty falls back to OTEL_* env only.
	Export OTLPExport
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
