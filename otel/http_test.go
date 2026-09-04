package otel

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

const (
	testTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	testSpanID  = "00f067aa0ba902b7"
	testSecret  = "test-internal-secret"
)

func testTraceparent() string {
	return "00-" + testTraceID + "-" + testSpanID + "-01"
}

func staticSecret(s string) func() string {
	return func() string { return s }
}

func TestTrustedInternalPropagator_ExtractHTTPHeader(t *testing.T) {
	p := TrustedInternalPropagator(propagation.TraceContext{}, HeaderInternalAuth, staticSecret(testSecret))
	tp := testTraceparent()

	// External request (no internal auth header) must not inherit the client trace.
	hExt := make(http.Header)
	hExt.Set("traceparent", tp)
	ctxExt := p.Extract(context.Background(), propagation.HeaderCarrier(hExt))
	if sc := trace.SpanContextFromContext(ctxExt); sc.IsValid() {
		t.Fatalf("external request should not inherit; got valid span context %v", sc)
	}

	// Internal request (header + matching secret) inherits the client trace.
	hInt := make(http.Header)
	hInt.Set("traceparent", tp)
	hInt.Set(HeaderInternalAuth, testSecret)
	ctxInt := p.Extract(context.Background(), propagation.HeaderCarrier(hInt))
	sc := trace.SpanContextFromContext(ctxInt)
	if !sc.IsValid() || sc.TraceID().String() != testTraceID {
		t.Fatalf("internal HTTP should inherit trace %s; got %v", testTraceID, sc)
	}

	// Header present but secret mismatch must not inherit.
	hBad := make(http.Header)
	hBad.Set("traceparent", tp)
	hBad.Set(HeaderInternalAuth, "wrong-secret")
	ctxBad := p.Extract(context.Background(), propagation.HeaderCarrier(hBad))
	if sc := trace.SpanContextFromContext(ctxBad); sc.IsValid() {
		t.Fatalf("wrong secret should not inherit; got %v", sc)
	}
}

func TestTrustedInternalPropagator_ExtractGRPCMetadata(t *testing.T) {
	p := TrustedInternalPropagator(propagation.TraceContext{}, HeaderInternalAuth, staticSecret(testSecret))
	carrier := propagation.MapCarrier{"traceparent": testTraceparent()}

	// Internal call: incoming MD carries the auth header with the matching secret.
	// Trust is detected via md.Get (keys are lowercase on the wire).
	ctxInt := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		HeaderInternalAuth, testSecret,
	))
	sc := trace.SpanContextFromContext(p.Extract(ctxInt, carrier))
	if !sc.IsValid() || sc.TraceID().String() != testTraceID {
		t.Fatalf("internal gRPC MD should allow extract; got %v", sc)
	}

	// Public call: no auth header in MD must not inherit.
	ctxPublic := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	if sc := trace.SpanContextFromContext(p.Extract(ctxPublic, carrier)); sc.IsValid() {
		t.Fatalf("public gRPC should not inherit; got %v", sc)
	}

	// Auth header present but secret mismatch must not inherit.
	ctxBad := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		HeaderInternalAuth, "wrong-secret",
	))
	if sc := trace.SpanContextFromContext(p.Extract(ctxBad, carrier)); sc.IsValid() {
		t.Fatalf("wrong secret should not inherit; got %v", sc)
	}
}

func TestTrustedInternalPropagator_EmptySecretTrustsNothing(t *testing.T) {
	// Empty secret = feature disabled: never trust anything (safe default),
	// even when a caller sends the header with any value.
	p := TrustedInternalPropagator(propagation.TraceContext{}, HeaderInternalAuth, staticSecret(""))
	carrier := propagation.MapCarrier{"traceparent": testTraceparent()}

	ctxInt := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		HeaderInternalAuth, "anything",
	))
	if sc := trace.SpanContextFromContext(p.Extract(ctxInt, carrier)); sc.IsValid() {
		t.Fatalf("empty secret must trust nothing; got %v", sc)
	}
}

func TestTrustedInternalPropagator_LiveSecretReload(t *testing.T) {
	cur := testSecret
	p := TrustedInternalPropagator(propagation.TraceContext{}, HeaderInternalAuth, func() string { return cur })
	tp := testTraceparent()

	h := make(http.Header)
	h.Set("traceparent", tp)
	h.Set(HeaderInternalAuth, testSecret)
	ctx := p.Extract(context.Background(), propagation.HeaderCarrier(h))
	if sc := trace.SpanContextFromContext(ctx); !sc.IsValid() {
		t.Fatal("matching live secret should inherit")
	}

	cur = "rotated-secret"
	ctx2 := p.Extract(context.Background(), propagation.HeaderCarrier(h))
	if sc := trace.SpanContextFromContext(ctx2); sc.IsValid() {
		t.Fatal("after rotation, old header value must not inherit")
	}
}
