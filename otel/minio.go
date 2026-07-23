package otel

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MinIOPlugin 提供 OTel 包装的 HTTP Transport，用于 MinIO/S3 调用追踪。
type MinIOPlugin struct {
	Config
}

func NewMinIOPlugin(cfg MinIOPlugin) *MinIOPlugin {
	return &cfg
}

// WrapTransport 包装 http.RoundTripper，为 S3 调用创建 client span。
func (p *MinIOPlugin) WrapTransport(base http.RoundTripper) http.RoundTripper {
	if p == nil || !p.Active() {
		return base
	}
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return "s3 " + r.Method + " " + r.URL.Path
		}),
	)
}
