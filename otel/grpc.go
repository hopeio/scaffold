package otel

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// GRPCPlugin 通过 otelgrpc StatsHandler 挂载。
type GRPCPlugin struct {
	Config
	Opts []otelgrpc.Option
}

func NewGRPCPlugin(cfg GRPCPlugin) *GRPCPlugin {
	return &cfg
}

func (p *GRPCPlugin) ServerHandler() stats.Handler {
	if p == nil || !p.Active() {
		return nil
	}
	return otelgrpc.NewServerHandler(p.Opts...)
}

func (p *GRPCPlugin) ClientHandler() stats.Handler {
	if p == nil || !p.Active() {
		return nil
	}
	return otelgrpc.NewClientHandler(p.Opts...)
}

func (p *GRPCPlugin) DialOption() grpc.DialOption {
	h := p.ClientHandler()
	if h == nil {
		return grpc.EmptyDialOption{}
	}
	return grpc.WithStatsHandler(h)
}

func (p *GRPCPlugin) ServerOption() grpc.ServerOption {
	h := p.ServerHandler()
	if h == nil {
		return grpc.EmptyServerOption{}
	}
	return grpc.StatsHandler(h)
}
