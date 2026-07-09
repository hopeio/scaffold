/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package errcode

import (
	"github.com/hopeio/mix"
)

type ErrCode = mix.ErrCode

const (
	Success ErrCode = mix.Success
	Canceled ErrCode = mix.Canceled
	Unknown ErrCode = mix.Unknown
	InvalidArgument ErrCode = mix.InvalidArgument
	NotFound ErrCode = mix.NotFound
	AlreadyExists ErrCode = mix.AlreadyExists
	PermissionDenied ErrCode = mix.PermissionDenied
	ResourceExhausted ErrCode = mix.ResourceExhausted
	FailedPrecondition ErrCode = mix.FailedPrecondition
	Aborted ErrCode = mix.Aborted
	OutOfRange ErrCode = mix.OutOfRange
	Unimplemented ErrCode = mix.Unimplemented
	Internal ErrCode = mix.Internal
	Unavailable ErrCode = mix.Unavailable
	DataLoss ErrCode = mix.DataLoss
	Unauthenticated ErrCode = mix.Unauthenticated
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


