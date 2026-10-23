package pwt_test

import (
	"testing"
	"time"

	"github.com/hopeio/scaffold/pwt"
)

func TestSignatureTakesTheLastSegment(t *testing.T) {
	if got := pwt.Signature("header.payload.sig"); got != "sig" {
		t.Fatalf("Signature = %q, want %q", got, "sig")
	}
	if got := pwt.Signature("  header.payload.sig  "); got != "sig" {
		t.Fatalf("Signature should trim spaces, got %q", got)
	}
}

func TestSignatureRejectsMalformedTokens(t *testing.T) {
	for _, token := range []string{"", "abc", "a.b", "a.b.c.d", "a.b."} {
		if got := pwt.Signature(token); got != "" {
			t.Fatalf("pwt.Signature(%q) = %q, want empty", token, got)
		}
	}
}

func TestTTLTracksRemainingLifetime(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got := pwt.TTL(now.Add(2*time.Hour), now)
	if got != 2*time.Hour+time.Second {
		t.Fatalf("TTL = %v, want just over 2h", got)
	}
}

func TestTTLIsZeroForAnExpiredToken(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	if got := pwt.TTL(now.Add(-time.Second), now); got != 0 {
		t.Fatalf("TTL = %v, want 0 for an expired token", got)
	}
}

func TestTTLFallsBackAndCaps(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	if got := pwt.TTL(time.Time{}, now); got != pwt.MaxTTL {
		t.Fatalf("TTL with no expiry = %v, want %v", got, pwt.MaxTTL)
	}
	if got := pwt.TTL(now.Add(365*24*time.Hour), now); got != pwt.MaxTTL {
		t.Fatalf("TTL = %v, want it capped at %v", got, pwt.MaxTTL)
	}
}

func TestKeyIsNamespaced(t *testing.T) {
	if got := pwt.Key("sig"); got != "user:token:revoked:sig" {
		t.Fatalf("Key = %q", got)
	}
}
