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

func TestNormalizeOTLPProtocol(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"  ":              "",
		"grpc":            "grpc",
		"GRPC":            "grpc",
		"http":            "http",
		"HTTP":            "http",
		"http/protobuf":   "http",
		"http-protobuf":   "http",
		"amqp":            "",
	}
	for in, want := range cases {
		if got := NormalizeOTLPProtocol(in); got != want {
			t.Errorf("NormalizeOTLPProtocol(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOTLPExportNormalize(t *testing.T) {
	in := OTLPExport{
		Endpoint:        "  http://base/  ",
		TracesEndpoint:  "\thttp://t/",
		MetricsEndpoint: " http://m/ ",
		LogsEndpoint:    " http://l/ ",
		Protocol:        " HTTP/Protobuf ",
		Headers: map[string]string{
			" Authorization ": " x ",
			"":               "drop",
			"k":              "",
		},
	}
	out := in.Normalize()
	if out.Endpoint != "http://base/" {
		t.Errorf("Endpoint = %q, want trimmed", out.Endpoint)
	}
	if out.Protocol != "http" {
		t.Errorf("Protocol = %q, want http", out.Protocol)
	}
	if v, ok := out.Headers["Authorization"]; !ok || v != "x" {
		t.Errorf("Headers = %v, want Authorization:x", out.Headers)
	}
	if _, ok := out.Headers[""]; ok {
		t.Errorf("empty header key should be dropped: %v", out.Headers)
	}
	if _, ok := out.Headers["k"]; ok {
		t.Errorf("empty header value should be dropped: %v", out.Headers)
	}
	// Normalize returns a copy; the input must be untouched.
	if in.Endpoint != "  http://base/  " {
		t.Errorf("Normalize mutated input: %q", in.Endpoint)
	}
}
