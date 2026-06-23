package wrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/types"
	gateway "github.com/hopeio/protobuf/tools/protoc-gen-gateway/gateway/gin"
)

func HandlerWrapCommon[REQ, RESP any](service types.Service[*REQ, *RESP]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := new(REQ)
		err := gateway.Bind(ctx, req)
		if err != nil {
			httpx.ServeError(ctx.Writer, ctx.Request, errors.InvalidArgument.Msg(err.Error()))
			return
		}

		res, reserr := service(ctx, req)
		if reserr != nil {
			httpx.ServeError(ctx.Writer, ctx.Request, reserr)
			return
		}
		if httpres, ok := any(res).(http.Handler); ok {
			httpres.ServeHTTP(ctx.Writer, ctx.Request)
			return
		}
		httpx.ServeSuccess(ctx.Writer, ctx.Request, res)
	}
}
