package cronlock

import (
	"testing"
	"time"
)

func TestTryRejectsEmptyInputs(t *testing.T) {
	if _, ok := Try(t.Context(), nil, "k", time.Minute); ok {
		t.Fatal("nil client must not acquire")
	}
	if _, ok := Try(t.Context(), nil, "", time.Minute); ok {
		t.Fatal("empty key must not acquire")
	}
	if _, ok := Try(t.Context(), nil, "k", 0); ok {
		t.Fatal("zero ttl must not acquire")
	}
}

func TestNewTokenIsUnique(t *testing.T) {
	a, b := newToken(), newToken()
	if a == "" || a == b {
		t.Fatalf("tokens must be non-empty and unique: %q %q", a, b)
	}
}
