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

type ErrCode errors.ErrCode

const (
	Success ErrCode = ErrCode(errors.Success)
	Canceled ErrCode = ErrCode(errors.Canceled)
	Unknown ErrCode = ErrCode(errors.Unknown)
	InvalidArgument ErrCode = ErrCode(errors.InvalidArgument)
	NotFound ErrCode = ErrCode(errors.NotFound)
	AlreadyExists ErrCode = ErrCode(errors.AlreadyExists)
	PermissionDenied ErrCode = ErrCode(errors.PermissionDenied)
	ResourceExhausted ErrCode = ErrCode(errors.ResourceExhausted)
	FailedPrecondition ErrCode = ErrCode(errors.FailedPrecondition)
	Aborted ErrCode = ErrCode(errors.Aborted)
	OutOfRange ErrCode = ErrCode(errors.OutOfRange)
	Unimplemented ErrCode = ErrCode(errors.Unimplemented)
	Internal ErrCode = ErrCode(errors.Internal)
	Unavailable ErrCode = ErrCode(errors.Unavailable)
	DataLoss ErrCode = ErrCode(errors.DataLoss)
	Unauthenticated ErrCode = ErrCode(errors.Unauthenticated)
	SysError ErrCode = 10000
	DBError ErrCode = 21000
	RowExists ErrCode = 21001
	RedisErr ErrCode = 22000
	IOError ErrCode = 30000
	UploadFail ErrCode = 30001
	UploadCheckFail ErrCode = 30002
	UploadCheckFormat ErrCode = 30003
	TimesTooMuch ErrCode = 30004
)

func init() {
	errors.Register(errors.ErrCode(SysError), "SysError")
	errors.Register(errors.ErrCode(DBError), "DBError")
	errors.Register(errors.ErrCode(RowExists), "RowExists")
	errors.Register(errors.ErrCode(RedisErr), "RedisErr")
	errors.Register(errors.ErrCode(IOError), "IOError")
	errors.Register(errors.ErrCode(UploadFail), "UploadFail")
	errors.Register(errors.ErrCode(UploadCheckFail), "UploadCheckFail")
	errors.Register(errors.ErrCode(UploadCheckFormat), "UploadCheckFormat")
	errors.Register(errors.ErrCode(TimesTooMuch), "TimesTooMuch")
}

func (x ErrCode) Code() int {
	return int(x)
}

func (x ErrCode) ErrResp() *errors.ErrResp {
	return errors.ErrCode(x).ErrResp()
}

func (x ErrCode) GRPCStatus() *status.Status {
	return status.New(codes.Code(x), errors.ErrCode(x).String())
}

func (x ErrCode) Msg(msg string) *errors.ErrResp {
	return &errors.ErrResp{Code: errors.ErrCode(x), Msg: msg}
}

func (x ErrCode) Wrap(err error) *errors.ErrResp {
	return &errors.ErrResp{Code: errors.ErrCode(x), Msg: err.Error()}
}

func (x ErrCode) Error() string {
	return errors.ErrCode(x).String()
}

/*func (x ErrCode) MarshalJSON() ([]byte, error) {
	return stringsx.ToBytes(`{"code":` + strconv.Itoa(int(x)) + `,"message":"` + x.String() + `"}`), nil
}

*/

