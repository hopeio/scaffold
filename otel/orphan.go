package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// SkipOrphanTracerProvider wraps tp so the tracers it hands out record nothing
// when ctx carries no span yet.
//
// Database/cache instrumentation hooks fire on paths the application never
// traces itself: pool dials, background reconnections, a ticker job's first
// Redis write. Without a parent each of those spans opens a trace containing
// exactly one span ("redis.dial", "redis.pipeline set"), which is both useless
// to read and the dominant trace count in a long-running process. Dropping them
// at the tracer keeps the SDK out of it entirely: no span object, no attributes,
// no export queue entry — instead of filtering the same spans in a sampler.
//
// A background job that does want its writes recorded starts a span first
// (StartJobSpan); the instrumentation then attaches underneath like it does for
// any request.
func SkipOrphanTracerProvider(tp trace.TracerProvider) trace.TracerProvider {
	return orphanProvider{TracerProvider: tp, resolve: func() trace.TracerProvider { return tp }}
}

// SkipOrphanGlobalProvider is SkipOrphanTracerProvider over the global provider,
// resolved per Tracer call. Plugins are mounted in dao.AfterInject, i.e. before
// SetupOTelSDK registers the SDK; the global provider is a delegate that swaps
// over once the SDK is up, so resolving lazily keeps that order harmless.
func SkipOrphanGlobalProvider() trace.TracerProvider {
	return orphanProvider{
		TracerProvider: otel.GetTracerProvider(),
		resolve:        otel.GetTracerProvider,
	}
}

// orphanProvider embeds a trace.TracerProvider because trace's interface is
// sealed (unexported method); the embedding supplies that method while Tracer
// below is the one every instrumentation actually calls.
type orphanProvider struct {
	trace.TracerProvider
	resolve func() trace.TracerProvider
}

func (p orphanProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return orphanTracer{Tracer: p.resolve().Tracer(name, opts...)}
}

type orphanTracer struct {
	trace.Tracer
}

// Start returns the no-op span of ctx when there is no parent to attach to. The
// caller keeps working unchanged: End, SetStatus and RecordError on a no-op span
// do nothing, and the returned ctx still carries no span, so a nested hook in the
// same orphan path is skipped by the same rule.
func (t orphanTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.Tracer.Start(ctx, name, opts...)
}
