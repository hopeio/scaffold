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

var protoOk, _ = proto.Marshal(&response.CommonResp{})
var marshalErr, _ = proto.Marshal(&response.CommonResp{
	Code: 14,
	Msg:  "failed to marshal error message",
})

func init() {
	gatewayx.Marshaler = &Protobuf{}
	gateway.HttpError = func(ctx *gin.Context, err error) {
		s, _ := status.FromError(err)
		delete(ctx.Request.Header, httpx.HeaderTrailer)
		ctx.Header("Grpc-Status", strconv.Itoa(int(s.Code())))
		message := &response.CommonResp{
			Code: int32(s.Code()),
			Msg:  s.Message(),
		}
		buf, err := gatewayx.Marshaler.Marshal(message)
		if err != nil {
			ctx.Header("Grpc-Status", "14")
			ctx.Header("Grpc-Message", "failed to marshal error message")
			return
		}
		ctx.Header(httpx.HeaderContentType, gatewayx.Marshaler.ContentType(message))
		ctx.Writer.Write(buf)
	}
	gateway.ForwardResponseMessage = func(ctx *gin.Context, md grpc.ServerMetadata, message proto.Message) {
		if !message.ProtoReflect().IsValid() {
			return
		}

		err := gatewayx.ForwardResponseMessage(ctx.Writer, ctx.Request, md, message, gatewayx.Marshaler)
		if err != nil {
			gateway.HttpError(ctx, err)
			return
		}
	}
}
