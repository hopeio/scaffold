package jwt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func TestSetOptions(t *testing.T) {
	Parser = jwt.NewParser()
	SetOptions(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "opt-u1"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var out Claims[*testAuth]
	if _, err := ParseToken(&out, token, secret); err != nil {
		t.Fatalf("ParseToken after SetOptions: %v", err)
	}
}

func TestGenerateToken_PackageLevel(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	now := time.Now()
	rc := jwt.RegisteredClaims{
		Subject:   "pkg-subject",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	token, err := GenerateToken(&rc, secret)
	if err != nil {
		t.Fatal(err)
	}
	var out jwt.RegisteredClaims
	if _, err := ParseToken(&out, token, secret); err != nil {
		t.Fatal(err)
	}
	if out.Subject != "pkg-subject" {
		t.Fatalf("subject=%q", out.Subject)
	}
}

func TestParseTokenWithKeyFunc_Success(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "keyfunc-u1"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var out Claims[*testAuth]
	_, err = ParseTokenWithKeyFunc(&out, token, func(_ *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Auth == nil || out.Auth.ID != "keyfunc-u1" {
		t.Fatalf("auth=%+v", out.Auth)
	}
}

func TestParseTokenWithKeyFunc_Failure(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	claims := NewClaims(&testAuth{ID: "keyfunc-u2"}, int64(time.Hour), "rfv")
	token, err := claims.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var out Claims[*testAuth]
	_, err = ParseTokenWithKeyFunc(&out, token, func(_ *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret-32-bytes-long!!!!!!"), nil
	})
	if err == nil {
		t.Fatal("want error for wrong key from key func")
	}

	_, err = ParseTokenWithKeyFunc(&out, "not-a-jwt", func(_ *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err == nil {
		t.Fatal("want error for malformed token")
	}
}

func TestParseToken_Expired(t *testing.T) {
	secret := []byte("secret-key-32-bytes-long!!!!!!!!")
	past := time.Now().Add(-time.Hour)
	c := &Claims[*testAuth]{
		Auth: &testAuth{ID: "expired-u1"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(past),
			IssuedAt:  jwt.NewNumericDate(past.Add(-time.Hour)),
			Issuer:    "rfv",
		},
	}
	token, err := c.GenerateToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	var out Claims[*testAuth]
	_, err = ParseToken(&out, token, secret)
	if err == nil {
		t.Fatal("want error for expired token")
	}
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("want jwt.ErrTokenExpired, got %v", err)
	}
}
