package gateway

import (
	"strconv"

	"github.com/gin-gonic/gin"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/grpc"
	gatewayx "github.com/hopeio/gox/net/http/grpc/gateway"
	"github.com/hopeio/protobuf/grpc/gateway"
	"github.com/hopeio/protobuf/response"
	"google.golang.org/grpc/grpclog"
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
		ctx.Header(httpx.HeaderContentType, gatewayx.Marshaler.ContentType(nil))
		ctx.Header("Grpc-Status", strconv.Itoa(int(s.Code())))
		se := &response.CommonResp{Code: uint32(s.Code()), Msg: s.Message()}
		buf, merr := gatewayx.Marshaler.Marshal(se)
		if merr != nil {
			grpclog.Infof("Failed to marshal error message %q: %v", se, merr)
			if _, err := ctx.Writer.Write(marshalErr); err != nil {
				grpclog.Infof("Failed to write response: %v", err)
			}
			return
		}

		if _, err := ctx.Writer.Write(buf); err != nil {
			grpclog.Infof("Failed to write response: %v", err)
		}
	}
	gateway.ForwardResponseMessage = func(ctx *gin.Context, md grpc.ServerMetadata, message proto.Message) {
		if !message.ProtoReflect().IsValid() {
			ctx.Writer.Write(protoOk)
			return
		}

		err := gatewayx.ForwardResponseMessage(ctx.Writer, ctx.Request, md, message, gatewayx.Marshaler)
		if err != nil {
			gateway.HttpError(ctx, err)
			return
		}
	}
}
