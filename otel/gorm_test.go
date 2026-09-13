package otel

import (
	"context"
	"testing"

	gormx "github.com/hopeio/gox/database/sql/gorm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestSlowSQLMetric_RecordAllDuration(t *testing.T) {
	reader := metric.NewManualReader()
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))
	defer otel.SetMeterProvider(prev)

	m := NewSlowSQLMetric(200)
	m.RecordAllDuration = true
	if err := m.Init(); err != nil {
		t.Fatal(err)
	}

	// Duration below threshold: only the all-query histogram should record.
	m.Record(&gormx.RecordContext{
		Ctx:        context.Background(),
		DurationMs: 5,
		Attrs:      []attribute.KeyValue{attribute.String("gorm.db.name", "test")},
	})

	var md metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &md); err != nil {
		t.Fatal(err)
	}
	got := findMetric(t, md, "gorm.db.query.duration_ms")
	hd, ok := got.Data.(metricdata.Histogram[float64])
	if !ok || len(hd.DataPoints) != 1 || hd.DataPoints[0].Count != 1 {
		t.Fatalf("gorm.db.query.duration_ms missing single data point: %#v", got.Data)
	}

	// A slow query must additionally bump the slow_sql counter and histogram,
	// while the all-query histogram keeps accumulating.
	m.Record(&gormx.RecordContext{
		Ctx:        context.Background(),
		DurationMs: 500,
		Attrs:      []attribute.KeyValue{attribute.String("gorm.db.name", "test")},
	})

	var md2 metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &md2); err != nil {
		t.Fatal(err)
	}
	all := findMetric(t, md2, "gorm.db.query.duration_ms")
	ahd := all.Data.(metricdata.Histogram[float64])
	if ahd.DataPoints[0].Count != 2 {
		t.Errorf("all-query histogram count = %d, want 2", ahd.DataPoints[0].Count)
	}
	slow := findMetric(t, md2, "gorm.db.slow_sql.requests")
	sc, _ := slow.Data.(metricdata.Sum[int64])
	if len(sc.DataPoints) == 0 || sc.DataPoints[0].Value != 1 {
		t.Errorf("gorm.db.slow_sql.requests = %#v, want 1", sc.DataPoints)
	}
}
