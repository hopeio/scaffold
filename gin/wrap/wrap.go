package wrap

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hopeio/gox/context/httpctx"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/gin/binding"
	"github.com/hopeio/gox/types"
)

func HandlerWrapGRPC[REQ, RES any](service types.GrpcService[*REQ, *RES]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := new(REQ)
		err := binding.Bind(ctx, req)
		if err != nil {
			ctx.JSON(http.StatusOK, errors.InvalidArgument.Wrap(err))
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
		ctx.JSON(http.StatusOK, httpx.NewSuccessRespData(res))
	}
}
