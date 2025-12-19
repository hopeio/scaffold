/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package errcode

import (
	"github.com/hopeio/gox/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (x ErrCode) Code() int {
	return int(x)
}

func (x ErrCode) ErrResp() *errors.ErrResp {
	return &errors.ErrResp{Code: errors.ErrCode(x), Msg: x.Comment()}
}

// example 实现
func (x ErrCode) GRPCStatus() *status.Status {
	return status.New(codes.Code(x), x.Comment())
}

func (x ErrCode) Msg(msg string) *errors.ErrResp {
	return &errors.ErrResp{Code: errors.ErrCode(x), Msg: msg}
}

func (x ErrCode) Wrap(err error) *errors.ErrResp {
	return &errors.ErrResp{Code: errors.ErrCode(x), Msg: err.Error()}
}

func (x ErrCode) Error() string {
	return x.Comment()
}

/*func (x ErrCode) MarshalJSON() ([]byte, error) {
	return stringsx.ToBytes(`{"code":` + strconv.Itoa(int(x)) + `,"message":"` + x.String() + `"}`), nil
}

*/

func init() {
	for code := range ErrCode_name {
		errors.Register(errors.ErrCode(code), ErrCode(code).Comment())
	}
}
