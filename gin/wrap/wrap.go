package wrap

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/context/httpctx"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/types"
	"github.com/hopeio/protobuf/grpc/gateway"
)

func HandlerWrapGRPC[REQ, RES any](service types.GrpcService[*REQ, *RES]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctxi := httpctx.FromRequest(httpctx.RequestCtx{ctx.Request, ctx.Writer})
		req := new(REQ)
		err := gateway.Bind(ctx, req)
		if err != nil {
			httpx.RespError(ctxi.Base(), ctx.Writer, errors.InvalidArgument.Msg(err.Error()))
			return
		}

		res, reserr := service(ctxi.Wrapper(), req)
		if reserr != nil {
			httpx.RespError(ctxi.Base(), ctx.Writer, reserr)
			return
		}
		if httpres, ok := any(res).(httpx.CommonResponder); ok {
			httpres.CommonRespond(ctxi.Base(), httpx.ResponseWriterWrapper{ctx.Writer})
			return
		}
		gateway.ForwardResponseMessage(ctx, ctx.Writer, res)
	}
}
