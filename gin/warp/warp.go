package warp

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/context/httpctx"
	"github.com/hopeio/utils/errors/errcode"
	httpi "github.com/hopeio/utils/net/http"
	"github.com/hopeio/utils/net/http/gin/binding"
	"github.com/hopeio/utils/types"
	"net/http"
)

func HandlerWrapCompatibleGRPC[REQ, RES any](service types.GrpcServiceMethod[*REQ, *RES]) gin.HandlerFunc {
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
		if httpres, ok := any(res).(httpi.IHttpResponse); ok {
			httpi.RespWrite(ctx.Writer, httpres)
			return
		}
		ctx.JSON(http.StatusOK, httpi.NewSuccessRespData(res))
	}
}
