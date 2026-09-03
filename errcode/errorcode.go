/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package errcode

import (
	"github.com/hopeio/mix"
	"google.golang.org/grpc/codes"
)

type ErrCode = mix.ErrCode

// 业务码布局见 mix.ErrCode 注释（业务码*100+gRPC code，十进制直读）：
// gRPC 通道上 status.code 只会是 <100 的标准枚举，业务码经 status.msg
// 与 ErrorInfo detail 传给客户端。
const (
	Success            ErrCode = mix.Success
	Canceled           ErrCode = mix.Canceled
	Unknown            ErrCode = mix.Unknown
	InvalidArgument    ErrCode = mix.InvalidArgument
	DeadlineExceeded   ErrCode = mix.DeadlineExceeded
	NotFound           ErrCode = mix.NotFound
	AlreadyExists      ErrCode = mix.AlreadyExists
	PermissionDenied   ErrCode = mix.PermissionDenied
	ResourceExhausted  ErrCode = mix.ResourceExhausted
	FailedPrecondition ErrCode = mix.FailedPrecondition
	Aborted            ErrCode = mix.Aborted
	OutOfRange         ErrCode = mix.OutOfRange
	Unimplemented      ErrCode = mix.Unimplemented
	Internal           ErrCode = mix.Internal
	Unavailable        ErrCode = mix.Unavailable
	DataLoss           ErrCode = mix.DataLoss
	Unauthenticated    ErrCode = mix.Unauthenticated
	SysError           ErrCode = 10000*100 + ErrCode(codes.Internal)
	DBError            ErrCode = 21000*100 + ErrCode(codes.Internal)
	RowExists          ErrCode = 21001*100 + ErrCode(codes.AlreadyExists)
	RedisErr           ErrCode = 22000*100 + ErrCode(codes.Internal)
	IOError            ErrCode = 30000*100 + ErrCode(codes.Internal)
	UploadFail         ErrCode = 30001*100 + ErrCode(codes.Internal)
	UploadCheckFail    ErrCode = 30002*100 + ErrCode(codes.InvalidArgument)
	UploadCheckFormat  ErrCode = 30003*100 + ErrCode(codes.InvalidArgument)
	TimesTooMuch       ErrCode = 30004*100 + ErrCode(codes.ResourceExhausted)
)

// init registers application-level error codes with human-readable names.
func init() {
	mix.RegisterErrCode(mix.ErrCode(SysError), "SysError")
	mix.RegisterErrCode(mix.ErrCode(DBError), "DBError")
	mix.RegisterErrCode(mix.ErrCode(RowExists), "RowExists")
	mix.RegisterErrCode(mix.ErrCode(RedisErr), "RedisErr")
	mix.RegisterErrCode(mix.ErrCode(IOError), "IOError")
	mix.RegisterErrCode(mix.ErrCode(UploadFail), "UploadFail")
	mix.RegisterErrCode(mix.ErrCode(UploadCheckFail), "UploadCheckFail")
	mix.RegisterErrCode(mix.ErrCode(UploadCheckFormat), "UploadCheckFormat")
	mix.RegisterErrCode(mix.ErrCode(TimesTooMuch), "TimesTooMuch")
}
