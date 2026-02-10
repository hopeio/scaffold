package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	contextx "github.com/hopeio/cherry"
)

type AuthInfo interface {
	GetId() string
}

type ClaimsWithRaw[A AuthInfo] struct {
	Claims[A]
	Raw []byte `json:"-"`
}

func (x *ClaimsWithRaw[A]) UnmarshalJSON(data []byte) error {
	x.Raw = data
	return json.Unmarshal(data, &x.Claims)
}

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

func Auth[A AuthInfo](ctx context.Context, secret []byte) (*Claims[A], error) {
	authorization := ClaimsWithRaw[A]{}
	metadata := contextx.GetMetadata(ctx)
	if metadata == nil {
		return nil, errors.New("no metadata")
	}
	if err := authorization.ParseToken(metadata.Token, secret); err != nil {
		return nil, err
	}
	metadata.AuthRaw = authorization.Raw
	metadata.AuthID = authorization.ID

	return &authorization.Claims, nil
}
