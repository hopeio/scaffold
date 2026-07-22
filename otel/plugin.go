package otel

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"strings"

	gormx "github.com/hopeio/gox/database/sql/gorm"
	redisotel "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
	"gorm.io/gorm"
)

// Plugin 通过各 client 原生插件/钩子挂载 OTel，不改动 initialize。
//
//	GORM  → db.Use(plugin)
//	Redis → redisotel.InstrumentTracing / InstrumentMetrics（go-redis Hook）
//	HTTP  → otelhttp.NewTransport / NewHandler
//	gRPC  → otelgrpc StatsHandler
type Plugin struct {
	Config Config
}

func NewPlugin(cfg Config) *Plugin { return &Plugin{Config: cfg} }

func Default() *Plugin { return NewPlugin(Config{}) }

func (p *Plugin) active() bool {
	if p == nil {
		return IsBootstrapped()
	}
	return p.Config.Active()
}

// GORM 使用 gorm.Plugin 钩子（Callback）。
func (p *Plugin) GORM(db *gorm.DB) error {
	if !p.active() || db == nil {
		return nil
	}
	slow := NewSlowSQLMetric(p.Config.SlowSQLMs)
	if err := slow.Init(); err != nil {
		return err
	}
	return db.Use(gormx.NewOTelPlugin(gormx.WithCustomMetrics(slow)))
}

// Redis 使用 go-redis Hook（InstrumentTracing / InstrumentMetrics）。
func (p *Plugin) Redis(client redis.UniversalClient) error {
	if !p.active() || client == nil {
		return nil
	}
	if err := redisotel.InstrumentTracing(client); err != nil {
		return err
	}
	return redisotel.InstrumentMetrics(client)
}

// HTTPTransport 包装 RoundTripper（otelhttp + ClientTrace）。
func (p *Plugin) HTTPTransport(base http.RoundTripper, opts ...otelhttp.Option) http.RoundTripper {
	if !p.active() {
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
	return otelhttp.NewTransport(base, append(defaults, opts...)...)
}

func (p *Plugin) HTTPClient(c *http.Client, opts ...otelhttp.Option) *http.Client {
	if !p.active() || c == nil {
		return c
	}
	c.Transport = p.HTTPTransport(c.Transport, opts...)
	return c
}

func (p *Plugin) HTTPHandler(h http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	if !p.active() || h == nil {
		return h
	}
	if operation == "" {
		operation = "http"
	}
	return otelhttp.NewHandler(h, operation, opts...)
}

func (p *Plugin) GRPCServerHandler(opts ...otelgrpc.Option) stats.Handler {
	if !p.active() {
		return nil
	}
	return otelgrpc.NewServerHandler(opts...)
}

func (p *Plugin) GRPCClientHandler(opts ...otelgrpc.Option) stats.Handler {
	if !p.active() {
		return nil
	}
	return otelgrpc.NewClientHandler(opts...)
}

func (p *Plugin) GRPCDialOption(opts ...otelgrpc.Option) grpc.DialOption {
	h := p.GRPCClientHandler(opts...)
	if h == nil {
		return grpc.EmptyDialOption{}
	}
	return grpc.WithStatsHandler(h)
}

func (p *Plugin) GRPCServerOption(opts ...otelgrpc.Option) grpc.ServerOption {
	h := p.GRPCServerHandler(opts...)
	if h == nil {
		return grpc.EmptyServerOption{}
	}
	return grpc.StatsHandler(h)
}

// SlowSQLMetric 作为 GORM CustomMetric 钩子。
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

func (m *SlowSQLMetric) Record(rc *gormx.RecordContext) {
	if rc == nil || rc.DurationMs < m.ThresholdMs {
		return
	}
	attrs := append(rc.Attrs, attribute.String("sql.verb", sqlVerb(rc)))
	opt := metric.WithAttributes(attrs...)
	m.counter.Add(rc.Ctx, 1, opt)
	m.histogram.Record(rc.Ctx, rc.DurationMs, opt)
}

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
