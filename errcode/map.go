package errcode

import (
	"context"
	"errors"

	"github.com/hopeio/gox/log"
	"github.com/hopeio/mix"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

// grpcStatus 与 mix.ErrCode / *mix.ErrResp 对齐：已类型化错误原样上抛。
type grpcStatus interface {
	GRPCStatus() *status.Status
}

// IsContextDone reports client cancel / deadline (not a storage fault).
func IsContextDone(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// LogError logs err at Error unless it is context cancel/deadline (then no-op).
func LogError(msg string, err error, fields ...zap.Field) {
	if err == nil || IsContextDone(err) {
		return
	}
	log.Errorw(msg, append(fields, zap.Error(err))...)
}

// Map 把 data 层原始错误收成对外 errcode：已是 GRPCStatus（含 ErrCode、*ErrResp）
// 则原样返回；context cancel/deadline 映射为 Canceled/DeadlineExceeded 且不打 ERROR；
// 否则打日志并返回 code（不带 Msg，不 Wrap）。
func Map(err error, code ErrCode) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(grpcStatus); ok {
		return err
	}
	if _, ok := err.(*mix.ErrResp); ok {
		return err
	}
	if errors.Is(err, context.Canceled) {
		return Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return DeadlineExceeded
	}
	log.Errorw(code.String(), zap.Error(err))
	return code
}
