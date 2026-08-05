package otel

import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/semconv/v1.21.0" // 请根据你的实际版本调整
)

// CustomSampler always samples requests with HTTP status >= 400 and samples others at 1%.
type CustomSampler struct{}

// ShouldSample forces sampling for error responses (HTTP status >= 400) and applies 1% ratio otherwise.
func (cs CustomSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
    for _, attr := range p.Attributes {
        if attr.Key == semconv.HTTPStatusCodeKey {
            if attr.Value.Type() == attribute.INT64 && attr.Value.AsInt64() >= 400 {
                return trace.SamplingResult{
                    Decision: trace.RecordAndSample,
                }
            }
        }
    }

    defaultSampler := trace.TraceIDRatioBased(0.01)
    return defaultSampler.ShouldSample(p)
}

func (cs CustomSampler) Description() string {
    return "CustomSampler"
}
