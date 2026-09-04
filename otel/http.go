package otel

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/http/httptrace"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
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
// (client trace_id/span_id) as attributes on the server root span for
// correlation/audit. Inheritance is controlled separately by the propagator
// (e.g. TrustedInternalPropagator): external callers stay on a fresh root;
// trusted internal callers may still inherit.
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

// HeaderInternalAuth is the shared header marking trusted internal
// service-to-service calls. HTTP and gRPC use the same name. The trust check
// must compare the secret **value**, not just header presence: a presence-only
// marker can be set by any client and gives no boundary. Each protocol
// normalizes case on the read side (net/http and grpc metadata.Get both
// lowercase), so this is the canonical mixed-case form. The gateway must
// convert inbound HTTP headers to gRPC metadata via metadata.New (lower-casing)
// so both links stay consistent.
const HeaderInternalAuth = "X-Internal-Auth"

// TrustedInternalPropagator wraps a real propagator so that ONLY requests
// carrying `header: secret` are trusted and have their incoming trace context
// extracted (inherited). Everything else is not extracted, so the server
// starts a fresh root span.
//
// The check compares the secret **value**, not just header presence: a
// presence-only marker can be set by any client and provides no boundary.
// An empty secret means nothing is trusted (safe default).
//
// Use this on HTTP ingress to mirror grpc's PublicEndpointFn behaviour.
func TrustedInternalPropagator(real propagation.TextMapPropagator, header, secret string) propagation.TextMapPropagator {
	if real == nil {
		real = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	return &trustedInternalPropagator{real: real, header: header, secret: secret}
}

type trustedInternalPropagator struct {
	real   propagation.TextMapPropagator
	header string
	secret string
}

// isInternalRequest reports whether the call carries the internal auth header
// with the matching secret.
//
// For HTTP (otelhttp), Extract runs with a HeaderCarrier before gRPC metadata
// exists on ctx — the header must be read from the carrier. For gRPC-shaped
// contexts, fall back to metadata.Get (keys are lowercased; never index MD
// with the mixed-case constant).
func (p *trustedInternalPropagator) isInternalRequest(ctx context.Context, carrier propagation.TextMapCarrier) bool {
	if p.header == "" || p.secret == "" {
		return false
	}
	var got string
	if carrier != nil {
		got = strings.TrimSpace(carrier.Get(p.header))
	}
	if got == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(p.header); len(vals) > 0 {
				got = strings.TrimSpace(vals[0])
			}
		}
	}
	return got != "" && internalAuthMatch(got, p.secret)
}

// internalAuthMatch 常量时间比较，避免通过响应时间侧信道逐字节猜密钥。
func internalAuthMatch(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (p *trustedInternalPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	if p.isInternalRequest(ctx, carrier) {
		return p.real.Extract(ctx, carrier)
	}
	return ctx
}

func (p *trustedInternalPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	p.real.Inject(ctx, carrier)
}

func (p *trustedInternalPropagator) Fields() []string {
	return p.real.Fields()
}
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
