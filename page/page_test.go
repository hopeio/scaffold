package page

import "testing"

func TestClampFillsInDefaults(t *testing.T) {
	no, size := Clamp(0, 0)
	if no != 1 || size != Default {
		t.Fatalf("Clamp(0, 0) = %d, %d; want 1, %d", no, size, Default)
	}
}

// pageSize 为 0 时 gormx 不生成 LIMIT，会把整张表拉回来。
func TestClampNeverReturnsAnUnboundedPage(t *testing.T) {
	if _, size := Clamp(3, 0); size == 0 {
		t.Fatal("Clamp returned pageSize 0, which means no LIMIT")
	}
}

func TestClampCapsOversizedPages(t *testing.T) {
	if _, size := Clamp(1, 100000); size != Max {
		t.Fatalf("Clamp pageSize = %d, want %d", size, Max)
	}
}

func TestClampKeepsReasonableValues(t *testing.T) {
	no, size := Clamp(4, 50)
	if no != 4 || size != 50 {
		t.Fatalf("Clamp(4, 50) = %d, %d", no, size)
	}
}

func TestClampIntGuardsAgainstNegatives(t *testing.T) {
	no, size := ClampInt(-5, -1)
	if no != 1 || size != int(Default) {
		t.Fatalf("ClampInt(-5, -1) = %d, %d; want 1, %d", no, size, Default)
	}
}
