package otel

import "testing"

func TestPyroscopeResolveEnabledFalseKeepsOff(t *testing.T) {
	t.Setenv("PYROSCOPE_SERVER_ADDRESS", "http://127.0.0.1:4040")
	c := PyroscopeConfig{Enabled: false, ServerAddress: "http://8.222.139.120:4040"}.resolve("hoper")
	if c.Enabled {
		t.Fatal("Enabled=false must not start pyroscope even when ServerAddress/env is set")
	}
	if c.ServerAddress != "http://8.222.139.120:4040" {
		t.Fatalf("address: %s", c.ServerAddress)
	}
}

func TestPyroscopeResolveEnabledFillsEnv(t *testing.T) {
	t.Setenv("PYROSCOPE_SERVER_ADDRESS", "http://127.0.0.1:4040")
	t.Setenv("PYROSCOPE_APPLICATION_NAME", "")
	c := PyroscopeConfig{Enabled: true}.resolve("my-svc")
	if !c.Enabled {
		t.Fatal("expected enabled")
	}
	if c.ServerAddress != "http://127.0.0.1:4040" {
		t.Fatalf("address: %s", c.ServerAddress)
	}
	if c.ApplicationName != "my-svc" {
		t.Fatalf("app: %s", c.ApplicationName)
	}
}
