package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	httpx "github.com/hopeio/gox/net/http"
)

type authorization[A httpx.Auth] struct {
	Claims[A]
	Raw []byte `json:"-"`
}

func (x *authorization[A]) UnmarshalJSON(data []byte) error {
	x.Raw = data
	return json.Unmarshal(data, &x.Claims)
}

func (x *authorization[A]) ParseToken(token string, secret []byte) error {
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

func Auth[A httpx.Auth](ctx context.Context, secret []byte) (*Claims[A], error) {
	authorization := authorization[A]{}
	ctxAuth := ctx.Auth()
	if err := authorization.ParseToken(ctxAuth.Token, secret); err != nil {
		return nil, err
	}
	ctxAuth.Raw = authorization.Raw
	ctxAuth.ID = authorization.ID
	ctxAuth.Info = authorization.Auth

	return &authorization.Claims, nil
}
