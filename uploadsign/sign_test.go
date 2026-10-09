package uploadsign

import (
	"strconv"
	"testing"
	"time"
)

const secret = "s3cr3t"

func TestVerifyRoundTrip(t *testing.T) {
	key := "image/2026/08/12/a.jpg"
	exp := time.Now().Add(time.Hour).Unix()
	sig := Sign(secret, key, exp)
	if sig == "" {
		t.Fatal("expected a signature")
	}
	if !Verify(secret, key, strconv.FormatInt(exp, 10), sig) {
		t.Fatal("valid signature rejected")
	}
	if Verify(secret, "image/2026/08/12/other.jpg", strconv.FormatInt(exp, 10), sig) {
		t.Fatal("signature must be bound to the key")
	}
	if Verify("other-secret", key, strconv.FormatInt(exp, 10), sig) {
		t.Fatal("signature must be bound to the secret")
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	key := "image/a.jpg"
	exp := time.Now().Add(-time.Minute).Unix()
	sig := Sign(secret, key, exp)
	if Verify(secret, key, strconv.FormatInt(exp, 10), sig) {
		t.Fatal("expired signature accepted")
	}
}

func TestQuerySkipsPublicPrefixes(t *testing.T) {
	if q := Query(secret, "public/splash.jpg", time.Hour, []string{"public/"}); q != "" {
		t.Fatalf("public objects must stay unsigned, got %q", q)
	}
	if q := Query(secret, "image/a.jpg", time.Hour, []string{"public/"}); q == "" {
		t.Fatal("private objects should be signed")
	}
}

func TestQueryDisabledWithoutSecret(t *testing.T) {
	if q := Query("", "image/a.jpg", time.Hour, nil); q != "" {
		t.Fatalf("no secret means no signature, got %q", q)
	}
	if Verify("", "image/a.jpg", "1", "x") {
		t.Fatal("verification must fail without a secret")
	}
}

func TestIsPlainMD5(t *testing.T) {
	if !IsPlainMD5("0123456789abcdef0123456789ABCDEF") {
		t.Fatal("hex md5 not recognised")
	}
	if IsPlainMD5("0123456789abcdef0123456789abcdef-3") {
		t.Fatal("multipart etag must not be treated as md5")
	}
	if IsPlainMD5("") || IsPlainMD5("zz") {
		t.Fatal("garbage accepted")
	}
}
