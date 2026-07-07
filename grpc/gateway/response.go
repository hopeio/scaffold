package gateway

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	httpx "github.com/hopeio/gox/net/http"
	gatewayx "github.com/hopeio/gox/net/http/grpc/gateway"
	gateway "github.com/hopeio/mix/gin"
	"github.com/hopeio/protobuf/response"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var protoOk, _ = proto.Marshal(&response.ErrResp{})
var marshalErr, _ = proto.Marshal(&response.ErrResp{
	Code: 14,
	Msg:  "failed to marshal error message",
})

func init() {
	gatewayx.DefaultMarshal = ProtobufMarshal
	gateway.HttpError = func(ctx *gin.Context, err error) {
		s, _ := status.FromError(err)
		delete(ctx.Request.Header, httpx.HeaderTrailer)
		errcode := strconv.Itoa(int(s.Code()))
		ctx.Header(httpx.HeaderGrpcStatus, errcode)
		ctx.Header(httpx.HeaderErrorCode, errcode)
		message := &response.ErrResp{
			Code: int32(s.Code()),
			Msg:  s.Message(),
		}

		buf, contentType := gatewayx.DefaultMarshal(ctx, message)

		ctx.Header(httpx.HeaderContentType, contentType)
		ow := ctx.Writer.(http.ResponseWriter)
		if uw, ok := ctx.Writer.(httpx.Unwrapper); ok {
			ow = uw.Unwrap()
		}
		if recorder, ok := ow.(httpx.RecordBodyer); ok {
			recorder.RecordBody(buf, message)
		}
		ctx.Writer.Write(buf)
	}
	gateway.HandleResponseMessage = func(ctx *gin.Context, message proto.Message) {
		if !message.ProtoReflect().IsValid() {
			return
		}

		err := gatewayx.HandleResponseMessage(ctx.Writer, ctx.Request, message)
		if err != nil {
			gateway.HttpError(ctx, err)
			return
		}
	}
}
