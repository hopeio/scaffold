package warp

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/context/httpctx"
	"github.com/hopeio/gox/errors/errcode"
	httpi "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/gin/binding"
	"github.com/hopeio/gox/types"
	"net/http"
)

func HandlerWrapCompatibleGRPC[REQ, RES any](service types.GrpcService[*REQ, *RES]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := new(REQ)
		err := binding.Bind(ctx, req)
		if err != nil {
			ctx.JSON(http.StatusOK, errcode.InvalidArgument.Wrap(err))
			return
		}
		ctxi := httpctx.FromRequest(httpctx.RequestCtx{ctx.Request, ctx.Writer})
		res, reserr := service(ctxi.Wrapper(), req)
		if reserr != nil {
			httpi.RespError(ctx.Writer, reserr)
			return
		}
		if httpres, ok := any(res).(httpi.ICommonResponseTo); ok {
			httpres.CommonResponse(httpi.CommonResponseWriter{ctx.Writer})
			return
		}
		ctx.JSON(http.StatusOK, httpi.NewSuccessRespData(res))
	}
}
