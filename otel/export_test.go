package otel

import "testing"

func TestOTLPExportSignalEndpoint(t *testing.T) {
	e := OTLPExport{
		Endpoint:        "http://base/",
		TracesEndpoint:  "http://t/",
		MetricsEndpoint: "http://m/",
		LogsEndpoint:    "http://l/",
		Protocol:        "http",
		Headers:         map[string]string{"Authorization": "x"},
	}
	if !e.configured() || e.useGRPC() {
		t.Fatal("http export")
	}
	if e.signalEndpoint("traces") != "http://t/" {
		t.Fatal(e.signalEndpoint("traces"))
	}
	if e.signalEndpoint("metrics") != "http://m/" {
		t.Fatal(e.signalEndpoint("metrics"))
	}
	if len(e.headers()) != 1 {
		t.Fatal(e.headers())
	}
}

func TestOTLPExportGRPC(t *testing.T) {
	e := OTLPExport{Protocol: "grpc"}
	if !e.useGRPC() {
		t.Fatal("grpc")
	}
}
