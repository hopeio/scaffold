package gateway

import (
	"strconv"

	"github.com/gin-gonic/gin"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/grpc"
	gatewayx "github.com/hopeio/gox/net/http/grpc/gateway"
	"github.com/hopeio/protobuf/grpc/gateway"
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
	gatewayx.DefaultMarshaler = &Protobuf{}
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
		contentType := gatewayx.DefaultMarshaler.ContentType(message)

		buf, err := gatewayx.DefaultMarshaler.Marshal(message)
		if err != nil {
			contentType = httpx.ContentTypeText
			ctx.Writer.Write([]byte(err.Error()))
		}
		ctx.Header(httpx.HeaderContentType, contentType)
		if ww, ok := ctx.Writer.(httpx.Unwrapper); ok {
			ow := ww.Unwrap()
			if recorder, ok := ow.(httpx.ResponseRecorder); ok {
				recorder.RecordResponse(contentType, buf, message)
			}
		}
		ctx.Writer.Write(buf)
	}
	gateway.ForwardResponseMessage = func(ctx *gin.Context, md grpc.ServerMetadata, message proto.Message) {
		if !message.ProtoReflect().IsValid() {
			return
		}

		err := gatewayx.ForwardResponseMessage(ctx.Writer, ctx.Request, md, message, gatewayx.DefaultMarshaler)
		if err != nil {
			gateway.HttpError(ctx, err)
			return
		}
	}
}
