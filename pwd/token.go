/*
 * Copyright 2026 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 */

// Package pwd signs and verifies three-segment tokens whose header and body are
// protobuf messages, using golang-jwt SigningMethod for the signature segment
// (same HS256/HS384/HS512 crypto as JWT, protobuf wire instead of JSON).
package pwd

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/protobuf/proto"
)

const Version = "1"

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpired      = errors.New("token is expired")
	ErrNotValidYet  = errors.New("token not valid yet")
	ErrAlgMismatch  = errors.New("unexpected signing method")
)

// DefaultMethod is used when Sign is called without an explicit method.
var DefaultMethod jwt.SigningMethod = jwt.SigningMethodHS256

// EncodeSegment base64url-encodes a segment with padding stripped (JWT style).
func EncodeSegment(seg []byte) string {
	return base64.RawURLEncoding.EncodeToString(seg)
}

// DecodeSegment base64url-decodes a JWT-style segment.
func DecodeSegment(seg string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(seg)
}

// Sign marshals header and body as protobuf, then signs with method (default HS256).
// Wire format: base64url(header).base64url(body).base64url(sig)
func Sign(header, body proto.Message, key []byte, method jwt.SigningMethod) (string, error) {
	if method == nil {
		method = DefaultMethod
	}
	hb, err := proto.Marshal(header)
	if err != nil {
		return "", err
	}
	bb, err := proto.Marshal(body)
	if err != nil {
		return "", err
	}
	sstr := EncodeSegment(hb) + "." + EncodeSegment(bb)
	sig, err := method.Sign(sstr, key)
	if err != nil {
		return "", err
	}
	return sstr + "." + EncodeSegment(sig), nil
}

// Parse splits, verifies the signature with key, and unmarshals header/body.
// exp/nbf are enforced when body implements Expirable.
func Parse(token string, key []byte, header, body proto.Message) error {
	return parse(token, key, header, body, true)
}

// ParseAllowExpired verifies the signature but does not enforce exp/nbf
// (e.g. reading exp for revocation TTL after the token already expired).
func ParseAllowExpired(token string, key []byte, header, body proto.Message) error {
	return parse(token, key, header, body, false)
}

func parse(token string, key []byte, header, body proto.Message, checkTimes bool) error {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return ErrInvalidToken
	}
	hb, err := DecodeSegment(parts[0])
	if err != nil {
		return ErrInvalidToken
	}
	bb, err := DecodeSegment(parts[1])
	if err != nil {
		return ErrInvalidToken
	}
	sig, err := DecodeSegment(parts[2])
	if err != nil {
		return ErrInvalidToken
	}
	if err := proto.Unmarshal(hb, header); err != nil {
		return ErrInvalidToken
	}
	alg := headerAlg(header)
	method := jwt.GetSigningMethod(alg)
	if method == nil {
		return ErrAlgMismatch
	}
	sstr := parts[0] + "." + parts[1]
	if err := method.Verify(sstr, sig, key); err != nil {
		return err
	}
	if err := proto.Unmarshal(bb, body); err != nil {
		return ErrInvalidToken
	}
	if checkTimes {
		return checkTime(body, time.Now())
	}
	return nil
}

type algGetter interface {
	GetAlg() string
}

func headerAlg(header proto.Message) string {
	if g, ok := header.(algGetter); ok {
		return g.GetAlg()
	}
	return ""
}

// Expirable is implemented by bodies that carry Unix exp/nbf seconds.
type Expirable interface {
	GetExp() int64
	GetNbf() int64
}

func checkTime(body proto.Message, now time.Time) error {
	e, ok := body.(Expirable)
	if !ok {
		return nil
	}
	unix := now.Unix()
	if exp := e.GetExp(); exp != 0 && unix >= exp {
		return ErrExpired
	}
	if nbf := e.GetNbf(); nbf != 0 && unix < nbf {
		return ErrNotValidYet
	}
	return nil
}
