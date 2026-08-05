package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// The helpers below are intended for clients without official hook/plugin support (e.g. NSQ, MQTT).
// Wrap calls at the business handler or publish site rather than modifying the initialize layer.

// StartClientSpan starts an OTel client-kind span with the given name and attributes.
func StartClientSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.Tracer(ScopeName).Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

// StartConsumerSpan starts an OTel consumer-kind span with the given name and attributes.
func StartConsumerSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.Tracer(ScopeName).Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

// EndSpan records the error (if any), sets the span status, and ends the span.
func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

// MessagingAttrs returns standard OTel semantic convention attributes for a messaging operation.
func MessagingAttrs(system, destination, operation string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.MessagingSystemKey.String(system),
		semconv.MessagingDestinationName(destination),
	}
	if operation != "" {
		attrs = append(attrs, semconv.MessagingOperationName(operation))
	}
	return attrs
}

// TraceClient wraps fn in a client span, ending it with the returned error.
func TraceClient(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartClientSpan(ctx, spanName, attrs...)
	err := fn(ctx)
	EndSpan(span, err)
	return err
}

// TraceConsumer wraps fn in a consumer span, ending it with the returned error.
func TraceConsumer(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartConsumerSpan(ctx, spanName, attrs...)
	err := fn(ctx)
	EndSpan(span, err)
	return err
}
