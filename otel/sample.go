package otel

import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/semconv/v1.21.0" // 请根据你的实际版本调整
)

// 自定义采样器：错误请求全采样，其他按 1% 采样
type CustomSampler struct{}

func (cs CustomSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
    // 1. 检查是否是根 Span，如果不是，通常继承父级决策
    // 这里简化处理，主要演示核心逻辑

    // 2. 检查 Span 属性，判断是否为错误请求
    // 注意：在实际应用中，需要确保在采样决策前属性已被设置
    for _, attr := range p.Attributes {
        if attr.Key == semconv.HTTPStatusCodeKey {
            if attr.Value.Type() == attribute.INT64 && attr.Value.AsInt64() >= 400 {
                return trace.SamplingResult{
                    Decision: trace.RecordAndSample,
                }
            }
        }
    }

    // 3. 非错误请求，按 1% 概率采样
    defaultSampler := trace.TraceIDRatioBased(0.01)
    return defaultSampler.ShouldSample(p)
}

func (cs CustomSampler) Description() string {
    return "CustomSampler"
}
