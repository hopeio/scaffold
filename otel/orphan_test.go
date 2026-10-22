package otel

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// The whole point of the wrapper is that a call with no parent produces nothing at
// all: no span object reaching the SDK, no export queue entry.
func TestSkipOrphanTracerProviderDropsParentlessSpans(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	sdk := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	tracer := SkipOrphanTracerProvider(sdk).Tracer("test")

	ctx, span := tracer.Start(context.Background(), "redis.dial")
	if span.IsRecording() {
		t.Fatal("parentless span must not record")
	}
	if span.SpanContext().IsValid() {
		t.Fatal("parentless span must not carry a span context")
	}
	// Instrumentation always ends its span; doing so must stay harmless.
	span.End()
	if got := len(rec.Ended()); got != 0 {
		t.Fatalf("ended spans = %d, want 0", got)
	}
	// The returned ctx keeps no span, so a nested hook in the same path is
	// dropped by the same rule instead of becoming a root of its own.
	if _, nested := tracer.Start(ctx, "redis.pipeline set"); nested.IsRecording() {
		t.Fatal("nested span under a dropped parent must not record")
	}
}

// Under a request (or a job span) the instrumentation keeps working as before.
func TestSkipOrphanTracerProviderKeepsChildSpans(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	sdk := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	tracer := SkipOrphanTracerProvider(sdk).Tracer("test")

	parentCtx, parent := sdk.Tracer("app").Start(context.Background(), "hub.keepalive")
	if !parent.IsRecording() {
		t.Fatal("the job span itself must record")
	}

	ctx, child := tracer.Start(parentCtx, "redis.pipeline set")
	if !child.IsRecording() {
		t.Fatal("span under a parent must record")
	}
	if !trace.SpanContextFromContext(ctx).IsValid() {
		t.Fatal("child ctx must carry the span for further nesting")
	}
	child.End()
	parent.End()

	ended := rec.Ended()
	if len(ended) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(ended))
	}
	if got := ended[0].SpanContext().TraceID(); got != parent.SpanContext().TraceID() {
		t.Fatalf("child landed in another trace: %v != %v", got, parent.SpanContext().TraceID())
	}
	if name := ended[0].Name(); name != "redis.pipeline set" {
		t.Fatalf("span name = %q", name)
	}
}

// Plugins are mounted in dao.AfterInject, i.e. while the global provider is still
// the delegate that SetupOTelSDK later swaps over. The wrapper must therefore ask
// for the provider per Tracer call rather than capturing it once at construction.
func TestOrphanProviderResolvesPerTracerCall(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	var current trace.TracerProvider = sdktrace.NewTracerProvider() // the pre-SDK state: no processor
	wrapped := orphanProvider{TracerProvider: current, resolve: func() trace.TracerProvider { return current }}

	sdk := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	current = sdk // the SDK is up now

	_, parent := sdk.Tracer("app").Start(context.Background(), "job")
	_, child := wrapped.Tracer("x").Start(trace.ContextWithSpan(context.Background(), parent), "redis.get")
	child.End()
	parent.End()
	if len(rec.Ended()) != 2 {
		t.Fatalf("ended spans = %d, want 2 (the wrapper kept the pre-SDK provider)", len(rec.Ended()))
	}
}
