package otel

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hopeio/scaffold/httproute"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsScrapePath is the request path prefix skipped by MetricsMiddleware
// (Prometheus-style scrape endpoint). Set to "" to disable the skip.
var MetricsScrapePath = "/metrics"

var (
	httpMetricsOnce    sync.Once
	httpRequestDur     metric.Float64Histogram
	httpRequestTotal   metric.Int64Counter
	httpRequestErrors  metric.Int64Counter
	httpActiveRequests metric.Int64UpDownCounter
)

func initHTTPMetrics() {
	meter := otel.GetMeterProvider().Meter(ScopeName)
	httpRequestDur, _ = meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("HTTP request duration in seconds"),
	)
	httpRequestTotal, _ = meter.Int64Counter(
		"http.server.request.total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	httpRequestErrors, _ = meter.Int64Counter(
		"http.server.request.errors",
		metric.WithDescription("Total number of HTTP requests with status >= 400"),
	)
	httpActiveRequests, _ = meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
	)
}

// statusRecorder wraps ResponseWriter to capture the status code.
// It embeds the interface (not a concrete type), so Flush/Hijack are not
// auto-promoted; Flush/Unwrap are forwarded explicitly. Without forwarding,
// gRPC ServeHTTP / SSE would fail with "requires http.Flusher".
type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// MetricsMiddleware records HTTP server business metrics: request duration,
// total, errors (status >= 400) and concurrent active requests. The route
// template (not the raw path) is used as the attribute to avoid cardinality
// blow-up from ids/md5s.
//
// A client disconnect/timeout (handler wrote nothing, context cancelled) is
// recorded as 499 so it is visible in metrics; nothing is written back to the
// already-closed client.
func MetricsMiddleware(next http.Handler) http.Handler {
	httpMetricsOnce.Do(initHTTPMetrics)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if MetricsScrapePath != "" && strings.HasPrefix(r.URL.Path, MetricsScrapePath) {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		ctx := r.Context()

		httpActiveRequests.Add(ctx, 1)
		defer httpActiveRequests.Add(ctx, -1)

		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		// Client cancelled/timed out: handler wrote nothing, rec.statusCode is
		// still the initial 200, which would be miscounted as success. Rewrite
		// to 499 so cancels/timeouts are visible. Never write to the client.
		if !rec.wroteHeader && r.Context().Err() != nil {
			rec.statusCode = 499
		}

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.statusCode)

		attrs := metric.WithAttributes(
			attribute.String("http.method", r.Method),
			// Route template, not the raw path, to avoid id/md5 exploding cardinality.
			attribute.String("http.route", httproute.Of(r)),
			attribute.String("http.status_code", status),
		)

		httpRequestDur.Record(ctx, duration, attrs)
		httpRequestTotal.Add(ctx, 1, attrs)

		if rec.statusCode >= 400 {
			httpRequestErrors.Add(ctx, 1, attrs)
		}
	})
}
