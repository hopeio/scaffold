package otel

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MinIOPlugin provides an OTel-instrumented HTTP Transport for tracing MinIO/S3 calls.
type MinIOPlugin struct {
	Config
}

// NewMinIOPlugin creates a MinIOPlugin from the given configuration.
func NewMinIOPlugin(cfg MinIOPlugin) *MinIOPlugin {
	return &cfg
}

// WrapTransport wraps base with an otelhttp transport that creates a client span for each S3 call.
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
