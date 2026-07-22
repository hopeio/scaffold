package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// 以下助手用于没有官方 Hook/Plugin 的客户端（NSQ/MQTT 等），
// 在业务 handler / publish 调用点手动包一层，而不是改 initialize。

func StartClientSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.Tracer(ScopeName).Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

func StartConsumerSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.Tracer(ScopeName).Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

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

func TraceClient(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartClientSpan(ctx, spanName, attrs...)
	err := fn(ctx)
	EndSpan(span, err)
	return err
}

func TraceConsumer(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartConsumerSpan(ctx, spanName, attrs...)
	err := fn(ctx)
	EndSpan(span, err)
	return err
}
