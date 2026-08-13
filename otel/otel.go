package otel

import (
	"context"
	"errors"
	"time"

	otelpyroscope "github.com/grafana/otel-profiling-go"
	"github.com/hopeio/gox/log"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
)

const ScopeName = "github.com/hopeio/scaffold"

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
// cfg.SampleRatio controls trace sampling; zero means never sample (set explicitly in config).
func SetupOTelSDK(ctx context.Context, res *resource.Resource, cfg SDKConfig) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Printf("[OTel SDK Error] %v", err)
	}))

	newPropagator()

	tracerProvider, stopProfiling, err := newTraceProvider(ctx, res, cfg)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	if stopProfiling != nil {
		shutdownFuncs = append(shutdownFuncs, stopProfiling)
	}

	// Set up meter provider.
	meterProvider, err := newMeterProvider(ctx, res, cfg)
	if err != nil {
		handleErr(err)
		return
	}

	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)

	loggerProvider, err := newLoggerProvider(ctx, res, cfg)
	if err != nil {
		handleErr(err)
		return
	}

	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)

	markBootstrapped()
	return
}

// newPropagator installs a composite W3C TraceContext+Baggage text-map propagator globally.
func newPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

// newTraceProvider creates and registers an OTLP HTTP trace provider using cfg.Sampler.
// When Pyroscope is enabled, wraps the provider with otel-profiling-go and starts pyroscope-go;
// stopProfiling (if non-nil) must be registered for shutdown.
func newTraceProvider(ctx context.Context, res *resource.Resource, cfg SDKConfig) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	var opts []otlptracehttp.Option
	if !cfg.Secure {
		// 显式 WithInsecure 会覆盖环境变量里的 https endpoint；Secure=true 时交还给 env/TLS
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	traceExporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(cfg.Sampler()),
	)

	pyroCfg := cfg.Pyroscope.resolve(serviceNameFromResource(res))
	if !pyroCfg.Enabled {
		otel.SetTracerProvider(tracerProvider)
		return tracerProvider, nil, nil
	}

	otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tracerProvider))
	profiler, err := startPyroscope(pyroCfg)
	if err != nil {
		_ = tracerProvider.Shutdown(ctx)
		return nil, nil, err
	}
	log.Infof("[OTel] pyroscope profiling → %s app=%s", pyroCfg.ServerAddress, pyroCfg.ApplicationName)
	return tracerProvider, func(context.Context) error { return profiler.Stop() }, nil
}

// newMeterProvider creates an OTLP HTTP meter provider with a periodic reader
// (cfg.MetricInterval, default 10s) and optional runtime metrics.
func newMeterProvider(ctx context.Context, res *resource.Resource, cfg SDKConfig) (*sdkmetric.MeterProvider, error) {
	var opts []otlpmetrichttp.Option
	if !cfg.Secure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}
	reader, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	interval := cfg.MetricInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(reader, sdkmetric.WithInterval(interval))),
	)
	otel.SetMeterProvider(meterProvider)
	if !cfg.DisableRuntimeMetrics {
		if err := runtime.Start(
			runtime.WithMeterProvider(meterProvider),
			runtime.WithMinimumReadMemStatsInterval(15*time.Second),
		); err != nil {
			// 库不该 Fatal 杀进程；采集失败交由调用方决定
			_ = meterProvider.Shutdown(ctx)
			return nil, err
		}
	}
	return meterProvider, nil
}

// newLoggerProvider creates an OTLP HTTP logger provider with batch log processing.
func newLoggerProvider(ctx context.Context, res *resource.Resource, cfg SDKConfig) (*sdklog.LoggerProvider, error) {
	var opts []otlploghttp.Option
	if !cfg.Secure {
		opts = append(opts, otlploghttp.WithInsecure())
	}
	exporter, err := otlploghttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(loggerProvider)
	return loggerProvider, nil
}
