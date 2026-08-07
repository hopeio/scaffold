package otel

import (
	"context"
	"errors"
	"log"
	"time"

	otelpyroscope "github.com/grafana/otel-profiling-go"
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

	tracerProvider, err := newTraceProvider(ctx, res, cfg)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)

	pyroCfg := cfg.Pyroscope.resolve(serviceNameFromResource(res))
	if pyroCfg.Enabled {
		otel.SetTracerProvider(otelpyroscope.NewTracerProvider(tracerProvider))
		profiler, perr := startPyroscope(pyroCfg)
		if perr != nil {
			handleErr(perr)
			return
		}
		shutdownFuncs = append(shutdownFuncs, func(context.Context) error {
			return profiler.Stop()
		})
		log.Printf("[OTel] pyroscope profiling → %s app=%s", pyroCfg.ServerAddress, pyroCfg.ApplicationName)
	}

	// Set up meter provider.
	meterProvider, err := newMeterProvider(ctx, res)
	if err != nil {
		handleErr(err)
		return
	}

	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)

	loggerProvider, err := newLoggerProvider(ctx, res)
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
func newTraceProvider(ctx context.Context, res *resource.Resource, cfg SDKConfig) (*sdktrace.TracerProvider, error) {
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure())
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(cfg.Sampler()),
	)
	otel.SetTracerProvider(tracerProvider)
	return tracerProvider, nil
}

// newMeterProvider creates an OTLP HTTP meter provider with a 10-second periodic reader and runtime metrics.
func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	reader, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, err
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(reader, sdkmetric.WithInterval(10*time.Second))),
	)
	otel.SetMeterProvider(meterProvider)
	if err := runtime.Start(
		runtime.WithMeterProvider(meterProvider),
		runtime.WithMinimumReadMemStatsInterval(15*time.Second),
	); err != nil {
		log.Fatalf("failed to start runtime instrumentation: %v", err)
	}
	return meterProvider, nil
}

// newLoggerProvider creates an OTLP HTTP logger provider with batch log processing.
func newLoggerProvider(ctx context.Context, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithInsecure())
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