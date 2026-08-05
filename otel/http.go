package otel

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPPlugin wraps HTTP Transport and Handler with OTel instrumentation via otelhttp.
type HTTPPlugin struct {
	Config
	Opts []otelhttp.Option
}

// NewHTTPPlugin creates an HTTPPlugin from the given configuration.
func NewHTTPPlugin(cfg HTTPPlugin) *HTTPPlugin {
	return &cfg
}

// Transport wraps base with OTel client-trace instrumentation; falls back to http.DefaultTransport when base is nil.
func (p *HTTPPlugin) Transport(base http.RoundTripper) http.RoundTripper {
	if p == nil || !p.Active() {
		return base
	}
	if base == nil {
		base = http.DefaultTransport
	}
	defaults := []otelhttp.Option{
		otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx)
		}),
	}
	return otelhttp.NewTransport(base, append(defaults, p.Opts...)...)
}

// Client instruments the given http.Client's Transport in place and returns it.
func (p *HTTPPlugin) Client(c *http.Client) *http.Client {
	if p == nil || !p.Active() || c == nil {
		return c
	}
	c.Transport = p.Transport(c.Transport)
	return c
}

// Handler wraps h with an OTel server span named by operation (defaults to "http").
func (p *HTTPPlugin) Handler(h http.Handler, operation string) http.Handler {
	if p == nil || !p.Active() || h == nil {
		return h
	}
	if operation == "" {
		operation = "http"
	}
	return otelhttp.NewHandler(h, operation, p.Opts...)
}
