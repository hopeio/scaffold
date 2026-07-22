package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// StartClientSpan 为出站 I/O（NSQ/MQTT/SMTP 等无官方 instrumentation 的客户端）开 span。
func StartClientSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.Tracer(ScopeName).Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

// StartConsumerSpan 为消息消费开 span。
func StartConsumerSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return otel.Tracer(ScopeName).Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attrs...),
	)
}

// EndSpan 结束 span 并按 err 设置状态。
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

// MessagingAttrs 构造 messaging 语义约定属性。
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

// TraceClient 包装一次客户端调用。
func TraceClient(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartClientSpan(ctx, spanName, attrs...)
	err := fn(ctx)
	EndSpan(span, err)
	return err
}

// TraceConsumer 包装一次消费处理。
func TraceConsumer(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartConsumerSpan(ctx, spanName, attrs...)
	err := fn(ctx)
	EndSpan(span, err)
	return err
}
