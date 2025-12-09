package wrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/context/httpctx"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/types"
	"github.com/hopeio/protobuf/grpc/gateway"
)

func HandlerWrapGRPC[REQ, RES any](service types.GrpcService[*REQ, *RES]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := new(REQ)
		err := gateway.Bind(ctx, req)
		if err != nil {
			data, err := gateway.Codec.Marshal(errors.InvalidArgument.Msg(err.Error()))
			if err != nil {
				ctx.Status(http.StatusInternalServerError)
				ctx.Abort()
				return
			}
			ctx.Writer.Write(data)
			return
		}
		ctxi := httpctx.FromRequest(httpctx.RequestCtx{ctx.Request, ctx.Writer})
		res, reserr := service(ctxi.Wrapper(), req)
		if reserr != nil {
			httpx.RespError(ctxi.Base(), ctx.Writer, reserr)
			return
		}
		if httpres, ok := any(res).(httpx.CommonResponder); ok {
			httpres.CommonRespond(ctxi.Base(), httpx.ResponseWriterWrapper{ctx.Writer})
			return
		}
		data, err := gateway.Codec.Marshal(httpx.NewSuccessRespData(res))
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
			ctx.Abort()
			return
		}
		ctx.Writer.Write(data)
	}
}
