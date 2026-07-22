package otel

import (
	"context"
	"net/http"
	"net/http/httptrace"

	gormx "github.com/hopeio/gox/database/sql/gorm"
	redisotel "github.com/redis/go-redis/extra/redisotel-native/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
	"gorm.io/gorm"
)

// Plugin 公共 I/O OTel 插件：对各客户端统一挂载 tracing/metrics。
type Plugin struct {
	Config Config
}

// NewPlugin 创建插件；cfg.Active() 为 false 时各 Instrument 方法为 no-op。
func NewPlugin(cfg Config) *Plugin {
	return &Plugin{Config: cfg}
}

// Default 使用默认 Config（跟随 SetupOTelSDK）。
func Default() *Plugin {
	return NewPlugin(Config{})
}

func (p *Plugin) active() bool {
	if p == nil {
		return IsBootstrapped()
	}
	return p.Config.Active()
}

// GORM 挂载 gox OTelPlugin + 慢 SQL metric。
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

// Redis 使用 redisotel-native，复用全局 MeterProvider（由 SetupOTelSDK 设置）。
func (p *Plugin) Redis(client redis.UniversalClient) error {
	if !p.active() || client == nil {
		return nil
	}
	inst := redisotel.GetObservabilityInstance()
	if inst.IsEnabled() {
		return nil
	}
	cfg := redisotel.NewConfig().WithEnabled(true).WithMeterProvider(otel.GetMeterProvider())
	return inst.Init(cfg)
}

// HTTPTransport 包装 RoundTripper（含 ClientTrace）。
func (p *Plugin) HTTPTransport(base http.RoundTripper, opts ...otelhttp.Option) http.RoundTripper {
	if !p.active() {
		return base
	}
	if base == nil {
		base = http.DefaultTransport
	}
	defaultOpts := []otelhttp.Option{
		otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
			return otelhttptrace.NewClientTrace(ctx)
		}),
	}
	return otelhttp.NewTransport(base, append(defaultOpts, opts...)...)
}

// HTTPClient 就地包装 http.Client.Transport。
func (p *Plugin) HTTPClient(c *http.Client, opts ...otelhttp.Option) *http.Client {
	if !p.active() || c == nil {
		return c
	}
	c.Transport = p.HTTPTransport(c.Transport, opts...)
	return c
}

// HTTPHandler 包装服务端 Handler。
func (p *Plugin) HTTPHandler(h http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	if !p.active() || h == nil {
		return h
	}
	if operation == "" {
		operation = "http"
	}
	return otelhttp.NewHandler(h, operation, opts...)
}

// GRPCServerHandler 返回 gRPC 服务端 StatsHandler。
func (p *Plugin) GRPCServerHandler(opts ...otelgrpc.Option) stats.Handler {
	if !p.active() {
		return nil
	}
	return otelgrpc.NewServerHandler(opts...)
}

// GRPCClientHandler 返回 gRPC 客户端 StatsHandler。
func (p *Plugin) GRPCClientHandler(opts ...otelgrpc.Option) stats.Handler {
	if !p.active() {
		return nil
	}
	return otelgrpc.NewClientHandler(opts...)
}

// GRPCDialOption 便于 Dial。
func (p *Plugin) GRPCDialOption(opts ...otelgrpc.Option) grpc.DialOption {
	h := p.GRPCClientHandler(opts...)
	if h == nil {
		return grpc.EmptyDialOption{}
	}
	return grpc.WithStatsHandler(h)
}

// GRPCServerOption 便于 NewServer。
func (p *Plugin) GRPCServerOption(opts ...otelgrpc.Option) grpc.ServerOption {
	h := p.GRPCServerHandler(opts...)
	if h == nil {
		return grpc.EmptyServerOption{}
	}
	return grpc.StatsHandler(h)
}
