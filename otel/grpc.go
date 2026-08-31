package otel

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// GRPCPlugin attaches OTel tracing to gRPC servers and clients via otelgrpc StatsHandlers.
type GRPCPlugin struct {
	Config
	Opts []otelgrpc.Option
}

// NewGRPCPlugin creates a GRPCPlugin from the given configuration.
func NewGRPCPlugin(cfg GRPCPlugin) *GRPCPlugin {
	return &cfg
}

// ServerHandler returns an otelgrpc server stats handler, or nil when the plugin is nil.
func (p *GRPCPlugin) ServerHandler() stats.Handler {
	if p == nil {
		return nil
	}
	return otelgrpc.NewServerHandler(p.Opts...)
}

// ClientHandler returns an otelgrpc client stats handler, or nil when the plugin is nil.
func (p *GRPCPlugin) ClientHandler() stats.Handler {
	if p == nil {
		return nil
	}
	return otelgrpc.NewClientHandler(p.Opts...)
}

// DialOption returns a grpc.DialOption that installs the OTel client stats handler.
func (p *GRPCPlugin) DialOption() grpc.DialOption {
	h := p.ClientHandler()
	if h == nil {
		return grpc.EmptyDialOption{}
	}
	return grpc.WithStatsHandler(h)
}

// ServerOption returns a grpc.ServerOption that installs the OTel server stats handler.
func (p *GRPCPlugin) ServerOption() grpc.ServerOption {
	h := p.ServerHandler()
	if h == nil {
		return grpc.EmptyServerOption{}
	}
	return grpc.StatsHandler(h)
}
