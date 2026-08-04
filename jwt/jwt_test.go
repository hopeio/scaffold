package jwt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hopeio/mix"
)

type testAuth struct {
	ID string `json:"id"`
}

func (a *testAuth) GetId() string { return a.ID }

func TestGenerateAndParseToken_RoundTrip(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "u1"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var out Claims[*testAuth]
	if _, err := ParseToken(&out, token, secret); err != nil {
		t.Fatal(err)
	}
	if out.Auth == nil || out.Auth.ID != "u1" {
		t.Fatalf("auth=%+v", out.Auth)
	}
	if out.Issuer != "rfv" {
		t.Fatalf("issuer=%q", out.Issuer)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "u1"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var out Claims[*testAuth]
	if _, err := ParseToken(&out, token, []byte("other-secret-32-bytes-long!!!!!!")); err == nil {
		t.Fatal("want error for wrong secret")
	}
}

func TestClaimsWithRaw_ParseToken_SetsIDAndRaw(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "uid-9"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var withRaw ClaimsWithRaw[*testAuth]
	if err := withRaw.ParseToken(token, secret); err != nil {
		t.Fatal(err)
	}
	if withRaw.ID != "uid-9" {
		t.Fatalf("ID(jti)=%q want uid-9", withRaw.ID)
	}
	if len(withRaw.Raw) == 0 {
		t.Fatal("Raw should be set via UnmarshalJSON")
	}
}

func TestClaimsWithRaw_NilAuth(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	// Generate token with nil pointer auth is awkward; build JWT without auth payload.
	c := &Claims[*testAuth]{}
	c.ExpiresAt = nil
	token, err := c.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var withRaw ClaimsWithRaw[*testAuth]
	err = withRaw.ParseToken(token, secret)
	if err == nil || err.Error() != "auth info is nil" {
		t.Fatalf("want auth info is nil, got %v", err)
	}
}

func TestAuth_RequiresMetadataAndSetsFields(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "auth-id"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Auth[*testAuth](context.Background(), token, secret)
	if err == nil || err.Error() != "no metadata" {
		t.Fatalf("want no metadata, got %v", err)
	}

	md := &mix.Metadata{}
	ctx := mix.WithMetadata(context.Background(), md)
	got, err := Auth[*testAuth](ctx, token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if got.Auth == nil || got.Auth.ID != "auth-id" {
		t.Fatalf("claims=%+v", got)
	}
	if md.AuthID != "auth-id" {
		t.Fatalf("AuthID=%q", md.AuthID)
	}
	if len(md.AuthRaw) == 0 {
		t.Fatal("AuthRaw empty")
	}
}

func TestAuth_EmptyToken(t *testing.T) {
	md := &mix.Metadata{}
	ctx := mix.WithMetadata(context.Background(), md)
	_, err := Auth[*testAuth](ctx, "", []byte("secret-key-32-bytes-long!!!!!!!!"))
	if err == nil {
		t.Fatal("want parse error for empty token")
	}
	if errors.Is(err, ErrInvalidToken) {
		// optional path
	}
}
