package otel

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// CustomSampler always samples requests with HTTP status >= 400 and samples others at the given ratio.
type CustomSampler struct {
	Ratio float64
}

// ShouldSample forces sampling for error responses (HTTP status >= 400) and applies Ratio otherwise.
func (cs CustomSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	for _, attr := range p.Attributes {
		if attr.Key == semconv.HTTPResponseStatusCodeKey {
			if attr.Value.Type() == attribute.INT64 && attr.Value.AsInt64() >= 400 {
				return trace.SamplingResult{
					Decision: trace.RecordAndSample,
				}
			}
		}
	}
	return ratioSampler(cs.Ratio).ShouldSample(p)
}

func (cs CustomSampler) Description() string {
	return "CustomSampler"
}
