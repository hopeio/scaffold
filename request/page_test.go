package param

import "testing"

func TestClampFillsInDefaults(t *testing.T) {
	no, size := Clamp(0, 0)
	if no != 1 || size != DefaultPageSize {
		t.Fatalf("Clamp(0, 0) = %d, %d; want 1, %d", no, size, DefaultPageSize)
	}
}

// gormx emits no LIMIT for pageSize 0, which would fetch the whole table.
func TestClampNeverReturnsAnUnboundedPage(t *testing.T) {
	if _, size := Clamp(3, 0); size == 0 {
		t.Fatal("Clamp returned pageSize 0, which means no LIMIT")
	}
}

func TestClampCapsOversizedPages(t *testing.T) {
	if _, size := Clamp(1, 100000); size != MaxPageSize {
		t.Fatalf("Clamp pageSize = %d, want %d", size, MaxPageSize)
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
	if no != 1 || size != int(DefaultPageSize) {
		t.Fatalf("ClampInt(-5, -1) = %d, %d; want 1, %d", no, size, DefaultPageSize)
	}
}
