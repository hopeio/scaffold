package otel

import (
	"context"
	"net/http"
	"testing"

	httpx "github.com/hopeio/gox/net/http"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

func TestTrustedInternalPropagator_ExtractHTTPHeader(t *testing.T) {
	p := TrustedInternalPropagator(propagation.TraceContext{})

	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	spanID := "00f067aa0ba902b7"
	tp := "00-" + traceID + "-" + spanID + "-01"

	hExt := make(http.Header)
	hExt.Set("traceparent", tp)
	ctxExt := p.Extract(context.Background(), propagation.HeaderCarrier(hExt))
	if sc := trace.SpanContextFromContext(ctxExt); sc.IsValid() {
		t.Fatalf("external request should not inherit; got valid span context %v", sc)
	}

	hInt := make(http.Header)
	hInt.Set("traceparent", tp)
	hInt.Set(httpx.HeaderGrpcInternal, httpx.HeaderGrpcInternal)
	ctxInt := p.Extract(context.Background(), propagation.HeaderCarrier(hInt))
	sc := trace.SpanContextFromContext(ctxInt)
	if !sc.IsValid() || sc.TraceID().String() != traceID {
		t.Fatalf("internal HTTP should inherit trace %s; got %v", traceID, sc)
	}
}

func TestTrustedInternalPropagator_ExtractGRPCMetadata(t *testing.T) {
	p := TrustedInternalPropagator(propagation.TraceContext{})

	traceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	spanID := "00f067aa0ba902b7"
	tp := "00-" + traceID + "-" + spanID + "-01"
	carrier := propagation.MapCarrier{"traceparent": tp}

	// Trust comes from incoming MD via md.Get (lowercase keys); carrier has no Grpc-Internal.
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		httpx.HeaderGrpcInternal, httpx.HeaderGrpcInternal,
	))
	ctxOut := p.Extract(ctx, carrier)
	sc := trace.SpanContextFromContext(ctxOut)
	if !sc.IsValid() || sc.TraceID().String() != traceID {
		t.Fatalf("internal gRPC MD should allow extract; got %v", sc)
	}

	ctxPublic := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	ctxOut = p.Extract(ctxPublic, carrier)
	if sc := trace.SpanContextFromContext(ctxOut); sc.IsValid() {
		t.Fatalf("public gRPC (no Grpc-Internal) should not inherit; got %v", sc)
	}
}
