package pwt_test

import (
	"bytes"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hopeio/scaffold/pwt"
)

func TestEncodeDecodeSegment(t *testing.T) {
	in := []byte{0, 1, 2, 250, 255}
	enc := pwt.EncodeSegment(in)
	out, err := pwt.DecodeSegment(enc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, out) {
		t.Fatalf("%v != %v", out, in)
	}
}

func TestHMACSigningStringMatchesJWTStyle(t *testing.T) {
	key := []byte("test-secret-key-32bytes-minimum!!")
	method := jwt.SigningMethodHS256
	sstr := pwt.EncodeSegment([]byte("hdr")) + "." + pwt.EncodeSegment([]byte("body"))
	sig, err := method.Sign(sstr, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := method.Verify(sstr, sig, key); err != nil {
		t.Fatal(err)
	}
	tok := sstr + "." + pwt.EncodeSegment(sig)
	if pwt.EncodeSegment(sig) != tok[len(sstr)+1:] {
		t.Fatal("segment join mismatch")
	}
}
