package otel

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPPlugin wraps HTTP Transport and Handler with OTel instrumentation via otelhttp.
type HTTPPlugin struct {
	Config
	Opts []otelhttp.Option
	// traceparentAttrs records the client-supplied W3C traceparent on the
	// server-generated root span as attributes (never inherits it).
	traceparentAttrs bool
}

// NewHTTPPlugin creates an HTTPPlugin from the given configuration.
func NewHTTPPlugin(cfg HTTPPlugin) *HTTPPlugin {
	return &cfg
}

// WithTraceparentAttributes enables recording the inbound W3C `traceparent`
// (client trace_id/span_id) as attributes on the server root span. The server
// always starts a fresh trace (combine with an empty propagator so it does not
// inherit the client span), keeping the client ids for correlation/audit only.
func (p *HTTPPlugin) WithTraceparentAttributes() *HTTPPlugin {
	if p != nil {
		p.traceparentAttrs = true
	}
	return p
}

// Transport wraps base with OTel client-trace instrumentation; falls back to http.DefaultTransport when base is nil.
func (p *HTTPPlugin) Transport(base http.RoundTripper) http.RoundTripper {
	if p == nil {
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
	if p == nil || c == nil {
		return c
	}
	c.Transport = p.Transport(c.Transport)
	return c
}

// Handler wraps h with an OTel server span named by operation (defaults to "http").
func (p *HTTPPlugin) Handler(h http.Handler, operation string) http.Handler {
	if p == nil || h == nil {
		return h
	}
	if operation == "" {
		operation = "http"
	}
	if p.traceparentAttrs {
		h = TraceparentAttributes(h)
	}
	return otelhttp.NewHandler(h, operation, p.Opts...)
}

// TraceparentAttributes records the client-supplied W3C `traceparent` as
// attributes on the server-generated root span, WITHOUT inheriting the client
// trace. The Go side always starts a fresh trace (otelhttp is configured with
// an empty propagator), so the client's trace_id/span_id are kept only for
// correlation/audit — never trusted as the authoritative parent.
func TraceparentAttributes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tp := r.Header.Get("traceparent"); tp != "" {
			traceID, spanID := parseTraceparent(tp)
			if traceID != "" {
				if span := trace.SpanFromContext(r.Context()); span != nil {
					attrs := []attribute.KeyValue{attribute.String("client.trace_id", traceID)}
					if spanID != "" {
						attrs = append(attrs, attribute.String("client.span_id", spanID))
					}
					span.SetAttributes(attrs...)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// parseTraceparent extracts trace_id (field 1) and span_id (field 2) from a W3C
// `traceparent` header: `version-trace_id-span_id-flags`. Returns "" for malformed input.
func parseTraceparent(tp string) (traceID, spanID string) {
	tp = strings.TrimSpace(tp)
	if tp == "" {
		return "", ""
	}
	// tolerate a possibly comma-separated extra value.
	if i := strings.IndexByte(tp, ','); i >= 0 {
		tp = tp[:i]
	}
	parts := strings.Split(tp, "-")
	if len(parts) < 3 {
		return "", ""
	}
	tid := parts[1]
	sid := parts[2]
	if len(tid) != 32 || len(sid) != 16 {
		return "", ""
	}
	if !isHex(tid) || !isHex(sid) {
		return "", ""
	}
	return tid, sid
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
