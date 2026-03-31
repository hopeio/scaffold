package otel

import (
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	gormx "github.com/hopeio/gox/database/sql/gorm"
)

type SlowSQLMetric struct {
	ThresholdMs float64
	counter     metric.Int64Counter
	histogram   metric.Float64Histogram
}

func NewSlowSQLMetric(thresholdMs float64) *SlowSQLMetric {
	if thresholdMs <= 0 {
		thresholdMs = 200
	}
	return &SlowSQLMetric{ThresholdMs: thresholdMs}
}

func (m *SlowSQLMetric) Init(meter metric.Meter) error {
	var err error
	m.counter, err = meter.Int64Counter("gorm.db.slow_sql.requests")
	if err != nil {
		return err
	}
	m.histogram, err = meter.Float64Histogram("gorm.db.slow_sql.duration_ms", metric.WithUnit("ms"))
	return err
}

func (m *SlowSQLMetric) Record(rc *gormx.RecordContext) {
	if rc == nil || rc.DurationMs < m.ThresholdMs {
		return
	}
	attrs := append(rc.Attrs, attribute.String("sql.verb", sqlVerb(rc)))
	attrOpt := metric.WithAttributes(attrs...)
	m.counter.Add(rc.Ctx, 1, attrOpt)
	m.histogram.Record(rc.Ctx, rc.DurationMs, attrOpt)
}

func sqlVerb(rc *gormx.RecordContext) string {
	if rc == nil || rc.Tx == nil || rc.Tx.Statement == nil {
		return ""
	}
	sql := strings.TrimSpace(rc.Tx.Statement.SQL.String())
	if sql == "" {
		return ""
	}
	for i := 0; i < len(sql); i++ {
		if sql[i] == ' ' || sql[i] == '\t' || sql[i] == '\n' {
			return strings.ToUpper(sql[:i])
		}
	}
	return strings.ToUpper(sql)
}
