package otel

import (
	"context"
	"net/http"
	"net/http/httptrace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPPlugin 通过 otelhttp 包装 Transport / Handler。
type HTTPPlugin struct {
	Config
	Opts []otelhttp.Option
}

func NewHTTPPlugin(cfg HTTPPlugin) *HTTPPlugin {
	return &cfg
}

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

func (p *HTTPPlugin) Client(c *http.Client) *http.Client {
	if p == nil || !p.Active() || c == nil {
		return c
	}
	c.Transport = p.Transport(c.Transport)
	return c
}

func (p *HTTPPlugin) Handler(h http.Handler, operation string) http.Handler {
	if p == nil || !p.Active() || h == nil {
		return h
	}
	if operation == "" {
		operation = "http"
	}
	return otelhttp.NewHandler(h, operation, p.Opts...)
}
