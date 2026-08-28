package otel

import (
	"context"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

func newTraceExporter(ctx context.Context, cfg SDKConfig) (sdktrace.SpanExporter, error) {
	e := cfg.Export
	if !e.configured() {
		return newTraceExporterHTTP(ctx, cfg, "")
	}
	if e.useGRPC() {
		return newTraceExporterGRPC(ctx, cfg, e.signalEndpoint("traces"))
	}
	return newTraceExporterHTTP(ctx, cfg, e.signalEndpoint("traces"))
}

func newMetricExporter(ctx context.Context, cfg SDKConfig) (sdkmetric.Exporter, error) {
	e := cfg.Export
	if !e.configured() {
		return newMetricExporterHTTP(ctx, cfg, "")
	}
	if e.useGRPC() {
		return newMetricExporterGRPC(ctx, cfg, e.signalEndpoint("metrics"))
	}
	return newMetricExporterHTTP(ctx, cfg, e.signalEndpoint("metrics"))
}

func newLogExporter(ctx context.Context, cfg SDKConfig) (sdklog.Exporter, error) {
	e := cfg.Export
	if !e.configured() {
		return newLogExporterHTTP(ctx, cfg, "")
	}
	if e.useGRPC() {
		return newLogExporterGRPC(ctx, cfg, e.signalEndpoint("logs"))
	}
	return newLogExporterHTTP(ctx, cfg, e.signalEndpoint("logs"))
}

func newTraceExporterHTTP(ctx context.Context, cfg SDKConfig, endpointURL string) (sdktrace.SpanExporter, error) {
	opts := otlpHTTPOpts(cfg, cfg.Export.headers(), endpointURL)
	return otlptracehttp.New(ctx, opts...)
}

func newTraceExporterGRPC(ctx context.Context, cfg SDKConfig, endpointURL string) (sdktrace.SpanExporter, error) {
	opts := otlpGRPCOpts(cfg, cfg.Export.headers(), endpointURL)
	return otlptracegrpc.New(ctx, opts...)
}

func newMetricExporterHTTP(ctx context.Context, cfg SDKConfig, endpointURL string) (sdkmetric.Exporter, error) {
	opts := otlpMetricHTTPOpts(cfg, cfg.Export.headers(), endpointURL)
	return otlpmetrichttp.New(ctx, opts...)
}

func newMetricExporterGRPC(ctx context.Context, cfg SDKConfig, endpointURL string) (sdkmetric.Exporter, error) {
	opts := otlpMetricGRPCOpts(cfg, cfg.Export.headers(), endpointURL)
	return otlpmetricgrpc.New(ctx, opts...)
}

func newLogExporterHTTP(ctx context.Context, cfg SDKConfig, endpointURL string) (sdklog.Exporter, error) {
	opts := otlpLogHTTPOpts(cfg, cfg.Export.headers(), endpointURL)
	return otlploghttp.New(ctx, opts...)
}

func newLogExporterGRPC(ctx context.Context, cfg SDKConfig, endpointURL string) (sdklog.Exporter, error) {
	opts := otlpLogGRPCOpts(cfg, cfg.Export.headers(), endpointURL)
	return otlploggrpc.New(ctx, opts...)
}

func otlpHTTPOpts(cfg SDKConfig, headers map[string]string, endpointURL string) []otlptracehttp.Option {
	var opts []otlptracehttp.Option
	if !cfg.Secure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(headers))
	}
	if endpointURL != "" {
		opts = append(opts, otlptracehttp.WithEndpointURL(endpointURL))
	}
	return opts
}

func otlpGRPCOpts(cfg SDKConfig, headers map[string]string, endpointURL string) []otlptracegrpc.Option {
	var opts []otlptracegrpc.Option
	if !cfg.Secure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(headers))
	}
	if endpointURL != "" {
		opts = append(opts, otlptracegrpc.WithEndpointURL(endpointURL))
	}
	return opts
}

func otlpMetricHTTPOpts(cfg SDKConfig, headers map[string]string, endpointURL string) []otlpmetrichttp.Option {
	var opts []otlpmetrichttp.Option
	if !cfg.Secure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(headers))
	}
	if endpointURL != "" {
		opts = append(opts, otlpmetrichttp.WithEndpointURL(endpointURL))
	}
	return opts
}

func otlpMetricGRPCOpts(cfg SDKConfig, headers map[string]string, endpointURL string) []otlpmetricgrpc.Option {
	var opts []otlpmetricgrpc.Option
	if !cfg.Secure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(headers))
	}
	if endpointURL != "" {
		opts = append(opts, otlpmetricgrpc.WithEndpointURL(endpointURL))
	}
	return opts
}

func otlpLogHTTPOpts(cfg SDKConfig, headers map[string]string, endpointURL string) []otlploghttp.Option {
	var opts []otlploghttp.Option
	if !cfg.Secure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(headers))
	}
	if endpointURL != "" {
		opts = append(opts, otlploghttp.WithEndpointURL(endpointURL))
	}
	return opts
}

func otlpLogGRPCOpts(cfg SDKConfig, headers map[string]string, endpointURL string) []otlploggrpc.Option {
	var opts []otlploggrpc.Option
	if !cfg.Secure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	if len(headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(headers))
	}
	if endpointURL != "" {
		opts = append(opts, otlploggrpc.WithEndpointURL(endpointURL))
	}
	return opts
}
