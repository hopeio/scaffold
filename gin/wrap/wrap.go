package wrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/context/httpctx"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	grpc_0 "github.com/hopeio/gox/net/http/grpc"
	"github.com/hopeio/gox/types"
	"github.com/hopeio/protobuf/grpc/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func HandlerWrapGRPC[REQ, RES any](service types.GrpcService[*REQ, *RES]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctxi := httpctx.FromRequest(httpctx.RequestCtx{ctx.Request, ctx.Writer})
		req := new(REQ)
		err := gateway.Bind(ctx, req)
		if err != nil {
			httpx.RespondError(ctx.Writer, ctx.Request, errors.InvalidArgument.Msg(err.Error()))
			return
		}
		var stream grpc_0.ServerTransportStream
		ctx.Request = ctx.Request.WithContext(grpc.NewContextWithServerTransportStream(metadata.NewIncomingContext(ctx.Request.Context(), metadata.MD(ctx.Request.Header)), &stream))

		res, reserr := service(ctxi.Wrapper(), req)
		if reserr != nil {
			httpx.RespondError(ctx.Writer, ctx.Request, reserr)
			return
		}
		if httpres, ok := any(res).(http.Handler); ok {
			httpres.ServeHTTP(ctx.Writer, ctx.Request)
			return
		}
		httpx.RespondSuccess(ctx.Writer, ctx.Request, res)
	}
}
