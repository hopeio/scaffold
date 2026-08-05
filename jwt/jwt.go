package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/hopeio/mix"
)

// AuthInfo is implemented by auth payloads that carry a string identity.
type AuthInfo interface {
	GetId() string
}

// ClaimsWithRaw embeds Claims and captures the raw JSON bytes during unmarshalling,
// allowing the original token payload to be stored in request metadata.
type ClaimsWithRaw[A AuthInfo] struct {
	Claims[A]
	Raw []byte `json:"-"`
}

// UnmarshalJSON saves the raw bytes before delegating to the embedded Claims.
func (x *ClaimsWithRaw[A]) UnmarshalJSON(data []byte) error {
	x.Raw = data
	return json.Unmarshal(data, &x.Claims)
}

// ParseToken validates the JWT string, populates the claims, and sets the auth ID.
func (x *ClaimsWithRaw[A]) ParseToken(token string, secret []byte) error {
	_, err := ParseToken(x, token, secret)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(x.Auth)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return errors.New("auth info is nil")
		}
	}
	x.ID = x.Auth.GetId()
	return nil
}

// Auth parses and validates the token, attaches the raw auth payload to the request metadata, and returns the typed claims.
func Auth[A AuthInfo](ctx context.Context, token string, secret []byte) (*Claims[A], error) {
	authorization := ClaimsWithRaw[A]{}
	metadata := mix.GetMetadata(ctx)
	if metadata == nil {
		return nil, errors.New("no metadata")
	}
	if err := authorization.ParseToken(token, secret); err != nil {
		return nil, err
	}
	metadata.AuthRaw = authorization.Raw
	metadata.AuthID = authorization.ID

	return &authorization.Claims, nil
}
