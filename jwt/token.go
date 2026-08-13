/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// Parser 默认锁定 HMAC 算法族做深度防御（本包签发一律 HS256，密钥为对称 []byte，
	// 不允许被换成其它算法家族）；业务可用 SetOptions(jwt.WithValidMethods(...)) 进一步收紧。
	Parser          = jwt.NewParser(jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
	ErrInvalidToken = errors.New("invalid token")
)

// SetOptions applies additional parser options to the global JWT Parser.
// 只应在启动早期（任何请求处理之前）调用：Parser 是无锁全局变量。
func SetOptions(options ...jwt.ParserOption) {
	for _, option := range options {
		option(Parser)
	}
}

// Claims holds typed authentication data alongside standard JWT registered claims.
// The Auth field should store immutable user info; refresh the token whenever the user changes credentials.
type Claims[T any] struct {
	Auth T `json:"auth,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken signs the claims with HS256 using the provided secret.
func (c *Claims[T]) GenerateToken(secret interface{}) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(secret)
}

// NewClaims creates a Claims value with the given auth data, expiry duration (as time.Duration nanoseconds), and issuer.
func NewClaims[T any](data T, maxAge int64, sign string) *Claims[T] {
	now := time.Now()
	exp := now.Add(time.Duration(maxAge))
	return &Claims[T]{
		Auth: data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{Time: exp},
			IssuedAt:  &jwt.NumericDate{Time: now},
			Issuer:    sign,
		},
	}
}

// GenerateToken signs arbitrary jwt.Claims with HS256 using the provided secret.
func GenerateToken(claims jwt.Claims, secret interface{}) (string, error) {
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString(secret)
	return token, err
}

// ParseToken parses and validates a JWT string, populating claims on success.
func ParseToken(claims jwt.Claims, token string, secret []byte) (*jwt.Token, error) {
	return Parser.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
}

// ParseTokenWithKeyFunc parses a JWT using a custom key-lookup function.
func ParseTokenWithKeyFunc(claims jwt.Claims, token string, f jwt.Keyfunc) (*jwt.Token, error) {
	return Parser.ParseWithClaims(token, claims, f)
}
