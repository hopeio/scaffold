package jwt

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/hopeio/gox/context/reqctx"
	"github.com/hopeio/gox/strings"
)

type authorization[A reqctx.AuthInfo] struct {
	Claims[A]
	AuthInfoRaw string `json:"-"`
}

func (x *authorization[A]) UnmarshalJSON(data []byte) error {
	x.AuthInfoRaw = strings.BytesToString(data)
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
	x.ID = x.Auth.IdStr()
	return nil
}

func Auth[REQ reqctx.ReqCtx, A reqctx.AuthInfo](ctx *reqctx.Context[REQ], secret []byte) (*Claims[A], error) {
	authorization := authorization[A]{}
	if err := authorization.ParseToken(ctx.Token, secret); err != nil {
		return nil, err
	}
	authInfo := authorization.Auth
	ctx.AuthID = authorization.ID
	ctx.AuthInfo = authInfo
	ctx.AuthInfoRaw = authorization.AuthInfoRaw
	return &authorization.Claims, nil
}
