package otel

import (
	"strings"

	gormx "github.com/hopeio/gox/database/sql/gorm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

// GORMPlugin attaches OTel tracing and slow-SQL metrics to a GORM DB via db.Use.
type GORMPlugin struct {
	Config
	SlowSQLMs float64 // slow SQL threshold in milliseconds; defaults to 200 when 0
}

// NewGORMPlugin creates a GORMPlugin from the given configuration.
func NewGORMPlugin(cfg GORMPlugin) *GORMPlugin {
	return &cfg
}

// Use calls db.Use to register the gox OTelPlugin together with the slow-SQL metric hook.
func (p *GORMPlugin) Use(db *gorm.DB) error {
	if p == nil || !p.Active() || db == nil {
		return nil
	}
	slow := NewSlowSQLMetric(p.SlowSQLMs)
	if err := slow.Init(); err != nil {
		return err
	}
	return db.Use(gormx.NewOTelPlugin(gormx.WithCustomMetrics(slow)))
}

type SlowSQLMetric struct {
	ThresholdMs float64
	counter     metric.Int64Counter
	histogram   metric.Float64Histogram
}

// NewSlowSQLMetric creates a SlowSQLMetric with the given threshold; defaults to 200 ms when ≤ 0.
func NewSlowSQLMetric(thresholdMs float64) *SlowSQLMetric {
	if thresholdMs <= 0 {
		thresholdMs = 200
	}
	return &SlowSQLMetric{ThresholdMs: thresholdMs}
}

// Init registers the slow-SQL counter and histogram instruments with the global meter.
func (m *SlowSQLMetric) Init() error {
	meter := otel.GetMeterProvider().Meter(ScopeName)
	var err error
	m.counter, err = meter.Int64Counter("gorm.db.slow_sql.requests")
	if err != nil {
		return err
	}
	m.histogram, err = meter.Float64Histogram("gorm.db.slow_sql.duration_ms", metric.WithUnit("ms"))
	return err
}

// Record emits counter and histogram measurements when the query duration exceeds the threshold.
func (m *SlowSQLMetric) Record(rc *gormx.RecordContext) {
	if rc == nil || rc.DurationMs < m.ThresholdMs {
		return
	}
	attrs := append(rc.Attrs, attribute.String("sql.verb", sqlVerb(rc)))
	opt := metric.WithAttributes(attrs...)
	m.counter.Add(rc.Ctx, 1, opt)
	m.histogram.Record(rc.Ctx, rc.DurationMs, opt)
}

// sqlVerb extracts the first SQL keyword (e.g. SELECT, INSERT) from the statement.
func sqlVerb(rc *gormx.RecordContext) string {
	if rc == nil || rc.DB == nil || rc.DB.Statement == nil {
		return ""
	}
	sql := strings.TrimSpace(rc.DB.Statement.SQL.String())
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
