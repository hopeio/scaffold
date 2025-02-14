/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package errcode

import (
	"github.com/hopeio/utils/errors/errcode"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (x ErrCode) Code() int {
	return int(x)
}

func (x ErrCode) ErrRep() *errcode.ErrRep {
	return &errcode.ErrRep{Code: errcode.ErrCode(x), Msg: x.String()}
}

// example 实现
func (x ErrCode) GRPCStatus() *status.Status {
	return status.New(codes.Code(x), x.String())
}

func (x ErrCode) Msg(msg string) *errcode.ErrRep {
	return &errcode.ErrRep{Code: errcode.ErrCode(x), Msg: msg}
}

func (x ErrCode) Wrap(err error) *errcode.ErrRep {
	return &errcode.ErrRep{Code: errcode.ErrCode(x), Msg: err.Error()}
}

func (x ErrCode) Error() string {
	return x.String()
}

func (x ErrCode) Origin() errcode.ErrCode {
	return errcode.ErrCode(x)
}

/*func (x ErrCode) MarshalJSON() ([]byte, error) {
	return stringsi.ToBytes(`{"code":` + strconv.Itoa(int(x)) + `,"message":"` + x.String() + `"}`), nil
}

*/

func init() {
	for code := range ErrCode_name {
		errcode.Register(errcode.ErrCode(code), ErrCode(code).Text())
	}
}
