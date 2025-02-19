package jwt

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hopeio/context/reqctx"
	stringsi "github.com/hopeio/utils/strings"
	jwti "github.com/hopeio/utils/validation/auth/jwt"
)

type authorization[A reqctx.AuthInfo] struct {
	Authorization[A]
	AuthInfoRaw string `json:"-"`
}

type Authorization[A reqctx.AuthInfo] struct {
	Auth A `json:"auth"`
	jwt.RegisteredClaims
}

func (x *Authorization[A]) GenerateToken(secret []byte) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, x).SignedString(secret)
}

func (x *authorization[A]) UnmarshalJSON(data []byte) error {
	x.AuthInfoRaw = stringsi.BytesToString(data)
	return json.Unmarshal(data, &x.Authorization)
}

func (x *authorization[A]) ParseToken(token string, secret []byte) error {
	_, err := jwti.ParseToken(x, token, secret)
	if err != nil {
		return err
	}
	x.ID = x.Auth.IdStr()
	return nil
}

func Auth[REQ reqctx.ReqCtx, A reqctx.AuthInfo](ctx *reqctx.Context[REQ], secret []byte) (*Authorization[A], error) {
	authorization := authorization[A]{}
	if err := authorization.ParseToken(ctx.Token, secret); err != nil {
		return nil, err
	}
	authInfo := authorization.Auth
	ctx.AuthID = authorization.ID
	ctx.AuthInfo = authInfo
	ctx.AuthInfoRaw = authorization.AuthInfoRaw
	return &authorization.Authorization, nil
}
