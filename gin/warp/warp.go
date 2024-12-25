package warp

import (
	"github.com/gin-gonic/gin"
	"github.com/hopeio/context/ginctx"
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
		ctxi := ginctx.FromRequest(ctx)
		res, reserr := service(ctxi.Wrapper(), req)
		if reserr != nil {
			if errcode, ok := err.(errcode.ErrCode); ok {
				httpi.RespErrcode(ctx.Writer, errcode)
				return
			}
			ctx.JSON(http.StatusOK, reserr)
			return
		}
		if httpres, ok := any(res).(httpi.IHttpResponse); ok {
			httpi.ResponseWrite(ctx.Writer, httpres)
			return
		}
		ctx.JSON(http.StatusOK, httpi.NewSuccessResData(res))
	}
}
