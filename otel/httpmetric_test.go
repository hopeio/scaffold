package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func findMetric(t *testing.T, md metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()
	for _, sm := range md.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Metrics{}
}

func hasAttr(attrs attribute.Set, key, val string) bool {
	for _, kv := range attrs.ToSlice() {
		if string(kv.Key) == key && kv.Value.AsString() == val {
			return true
		}
	}
	return false
}

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	reader := metric.NewManualReader()
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))
	httpMetricsOnce = sync.Once{}
	defer func() {
		otel.SetMeterProvider(prev)
	}()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := MetricsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var md metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &md); err != nil {
		t.Fatal(err)
	}

	total := findMetric(t, md, "http.server.request.total")
	sum, ok := total.Data.(metricdata.Sum[int64])
	if !ok || len(sum.DataPoints) == 0 {
		t.Fatalf("http.server.request.total has no data points: %#v", total.Data)
	}
	if sum.DataPoints[0].Value != 1 {
		t.Errorf("total = %d, want 1", sum.DataPoints[0].Value)
	}
	if !hasAttr(sum.DataPoints[0].Attributes, "http.status_code", "418") {
		t.Errorf("total missing status_code=418 attr: %v", sum.DataPoints[0].Attributes)
	}
	if !hasAttr(sum.DataPoints[0].Attributes, "http.method", "GET") {
		t.Errorf("total missing method=GET attr: %v", sum.DataPoints[0].Attributes)
	}

	// duration histogram must also have a data point.
	dur := findMetric(t, md, "http.server.request.duration")
	if hd, ok := dur.Data.(metricdata.Histogram[float64]); !ok || len(hd.DataPoints) == 0 {
		t.Errorf("http.server.request.duration has no data points: %#v", dur.Data)
	}

	// 4xx must increment the errors counter.
	errs := findMetric(t, md, "http.server.request.errors")
	es, ok := errs.Data.(metricdata.Sum[int64])
	if !ok || len(es.DataPoints) == 0 || es.DataPoints[0].Value != 1 {
		t.Errorf("http.server.request.errors = %#v, want 1", errs.Data)
	}
}

func TestMetricsMiddleware_ClientCancelIs499(t *testing.T) {
	reader := metric.NewManualReader()
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))
	httpMetricsOnce = sync.Once{}
	defer func() {
		otel.SetMeterProvider(prev)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a client disconnect mid-handler: handler writes nothing.
		cancel()
	})
	h := MetricsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)
	h.ServeHTTP(httptest.NewRecorder(), req)

	var md metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &md); err != nil {
		t.Fatal(err)
	}
	total := findMetric(t, md, "http.server.request.total")
	sum := total.Data.(metricdata.Sum[int64])
	if len(sum.DataPoints) == 0 || !hasAttr(sum.DataPoints[0].Attributes, "http.status_code", "499") {
		t.Fatalf("expected a 499 data point, got %#v", sum.DataPoints)
	}
}

func TestMetricsMiddleware_SkipsScrapePath(t *testing.T) {
	MetricsScrapePath = "/metrics"
	reader := metric.NewManualReader()
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))
	httpMetricsOnce = sync.Once{}
	defer func() {
		otel.SetMeterProvider(prev)
	}()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	h := MetricsMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	var md metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &md); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("next handler not called for scrape path")
	}
	if len(md.ScopeMetrics) != 0 {
		// No metrics should be emitted for the skipped path.
		t.Errorf("scrape path should emit no metrics, got %#v", md.ScopeMetrics)
	}
}
