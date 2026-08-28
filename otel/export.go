package otel

import "strings"

// OTLPExport configures OTLP exporter endpoints, protocol, and outbound headers.
// hoper loads [OTel.Export] from config; App/Agent use ClientBootstrap.OTel.
type OTLPExport struct {
	Endpoint        string
	TracesEndpoint  string
	MetricsEndpoint string
	LogsEndpoint    string
	// Protocol: http | grpc; empty defaults to http.
	Protocol string
	Headers  map[string]string
}

func (e OTLPExport) configured() bool {
	return strings.TrimSpace(e.Endpoint) != "" ||
		strings.TrimSpace(e.TracesEndpoint) != "" ||
		strings.TrimSpace(e.MetricsEndpoint) != "" ||
		strings.TrimSpace(e.LogsEndpoint) != "" ||
		len(e.headers()) > 0 ||
		strings.TrimSpace(e.Protocol) != ""
}

func (e OTLPExport) useGRPC() bool {
	return strings.EqualFold(strings.TrimSpace(e.Protocol), "grpc")
}

func (e OTLPExport) signalEndpoint(signal string) string {
	switch signal {
	case "traces":
		if s := strings.TrimSpace(e.TracesEndpoint); s != "" {
			return s
		}
	case "metrics":
		if s := strings.TrimSpace(e.MetricsEndpoint); s != "" {
			return s
		}
	case "logs":
		if s := strings.TrimSpace(e.LogsEndpoint); s != "" {
			return s
		}
	}
	return strings.TrimSpace(e.Endpoint)
}

func (e OTLPExport) headers() map[string]string {
	if len(e.Headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(e.Headers))
	for k, v := range e.Headers {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
