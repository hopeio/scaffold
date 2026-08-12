package otel

import (
	"os"
	"runtime"
	"strings"

	"github.com/grafana/pyroscope-go"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// PyroscopeConfig enables continuous profiling (pyroscope-go) and optional
// span↔profile correlation via otel-profiling-go when ServerAddress is set.
type PyroscopeConfig struct {
	Enabled             bool
	ServerAddress       string
	ApplicationName     string
	BasicAuthUser       string
	BasicAuthPassword   string
	AuthToken           string
	DisableMutexProfile bool
	DisableBlockProfile bool
}

func (c PyroscopeConfig) resolve(appName string) PyroscopeConfig {
	out := c
	if out.ServerAddress == "" {
		out.ServerAddress = strings.TrimSpace(os.Getenv("PYROSCOPE_SERVER_ADDRESS"))
	}
	if out.ApplicationName == "" {
		out.ApplicationName = strings.TrimSpace(os.Getenv("PYROSCOPE_APPLICATION_NAME"))
	}
	if out.ApplicationName == "" {
		out.ApplicationName = appName
	}
	if out.BasicAuthUser == "" {
		out.BasicAuthUser = os.Getenv("PYROSCOPE_BASIC_AUTH_USER")
	}
	if out.BasicAuthPassword == "" {
		out.BasicAuthPassword = os.Getenv("PYROSCOPE_BASIC_AUTH_PASSWORD")
	}
	if out.AuthToken == "" {
		out.AuthToken = os.Getenv("PYROSCOPE_AUTH_TOKEN")
	}
	// Enabled 是总开关：false 即使配了 ServerAddress / PYROSCOPE_SERVER_ADDRESS 也不启动。
	return out
}

func serviceNameFromResource(res *resource.Resource) string {
	if res == nil {
		return ""
	}
	for _, a := range res.Attributes() {
		if a.Key == semconv.ServiceNameKey {
			return a.Value.AsString()
		}
	}
	return ""
}

func startPyroscope(cfg PyroscopeConfig) (*pyroscope.Profiler, error) {
	if !cfg.DisableMutexProfile {
		runtime.SetMutexProfileFraction(5)
	}
	if !cfg.DisableBlockProfile {
		runtime.SetBlockProfileRate(5)
	}
	pc := pyroscope.Config{
		ApplicationName: cfg.ApplicationName,
		ServerAddress:   cfg.ServerAddress,
		Logger:          nil, // quiet by default; set StandardLogger via env if needed
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	}
	if cfg.AuthToken != "" {
		pc.AuthToken = cfg.AuthToken
	} else if cfg.BasicAuthUser != "" {
		pc.BasicAuthUser = cfg.BasicAuthUser
		pc.BasicAuthPassword = cfg.BasicAuthPassword
	}
	if os.Getenv("PYROSCOPE_LOG") == "1" {
		pc.Logger = pyroscope.StandardLogger
	}
	return pyroscope.Start(pc)
}
