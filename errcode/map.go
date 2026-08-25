package errcode

import (
	"github.com/hopeio/gox/log"
	"github.com/hopeio/mix"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

// grpcStatus 与 mix.ErrCode / *mix.ErrResp 对齐：已类型化错误原样上抛。
type grpcStatus interface {
	GRPCStatus() *status.Status
}

// Map 把 data 层原始错误收成对外 errcode：已是 GRPCStatus（含 ErrCode、*ErrResp）
// 则原样返回；否则打日志并返回 code（不带 Msg，不 Wrap）。
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
	log.Errorw(code.String(), zap.Error(err))
	return code
}
