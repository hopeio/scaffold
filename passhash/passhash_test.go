package passhash

import "testing"

func TestArgonRoundTrip(t *testing.T) {
	h := Hash("secret-pass", "pepper")
	if !stringsHasPrefix(h, argonPrefix) {
		t.Fatalf("hash %q", h)
	}
	if !Verify("secret-pass", h, "pepper") {
		t.Fatal("verify failed")
	}
	if Verify("wrong-pass", h, "pepper") {
		t.Fatal("wrong password accepted")
	}
	if Verify("secret-pass", h, "other") {
		t.Fatal("wrong pepper accepted")
	}
	if Verify("secret-pass", "d41d8cd98f00b204e9800998ecf8427e", "pepper") {
		t.Fatal("legacy non-argon hash must be rejected")
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
