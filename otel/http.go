package otel

import (
	"context"
	"crypto/subtle"
	"net/http"
	"net/http/httptrace"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
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
// An empty secret (or nil secret fn) means nothing is trusted (safe default).
//
// secret is called on each Extract so hot-reloaded config stays in sync with
// business-side IsInternalCall checks that also read the live secret.
//
// Use this on HTTP ingress to mirror grpc's PublicEndpointFn behaviour.
func TrustedInternalPropagator(real propagation.TextMapPropagator, header string, secret func() string) propagation.TextMapPropagator {
	if real == nil {
		real = propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	return &trustedInternalPropagator{real: real, header: header, secret: secret}
}

type trustedInternalPropagator struct {
	real   propagation.TextMapPropagator
	header string
	secret func() string
}

// isInternalRequest reports whether the call carries the internal auth header
// with the matching secret.
//
// For HTTP (otelhttp), Extract runs with a HeaderCarrier before gRPC metadata
// exists on ctx — the header must be read from the carrier. For gRPC-shaped
// contexts, fall back to metadata.Get (keys are lowercased; never index MD
// with the mixed-case constant).
func (p *trustedInternalPropagator) isInternalRequest(ctx context.Context, carrier propagation.TextMapCarrier) bool {
	if p.header == "" || p.secret == nil {
		return false
	}
	want := strings.TrimSpace(p.secret())
	if want == "" {
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
	return got != "" && internalAuthMatch(got, want)
}

// internalAuthMatch is a constant-time compare to avoid timing side channels.
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
