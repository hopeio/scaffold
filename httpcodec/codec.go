// Package httpcodec 把 Mix 网关改成 protobuf 优先、JSON 备选（看 Accept）。
//
// HTTP 错误约定：JSON 体 {code,msg,data}（msg 为 i18n key，data 为占位符 map）；
// protobuf 靠 Error-Code / Grpc-Status 头判断出错，体为 google.rpc.ErrorInfo。
package httpcodec

import (
	"context"
	"net/http"
	"strings"

	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/mix"
	"github.com/hopeio/mix/gateway"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/protobuf/proto"
)

// Install 覆盖 mix 默认 JSON marshal 与网关响应/错误写出。
func Install() {
	mix.DefaultMarshal = marshalPreferPB
	gateway.HandleResponseMessage = handleResponse
	gateway.HandleError = handleError
}

func wantsJSON(r *http.Request) bool {
	if r == nil {
		return false
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	if strings.Contains(accept, "protobuf") {
		return false
	}
	return strings.Contains(accept, "application/json") || strings.Contains(accept, "text/json")
}

func marshalPreferPB(ctx context.Context, v any) ([]byte, string, error) {
	if md := mix.GetMetadata(ctx); md != nil && wantsJSON(md.Request) {
		return gateway.JsonMarshal(ctx, v)
	}
	return marshalPB(ctx, v)
}

func marshalForRequest(r *http.Request, v any) ([]byte, string, error) {
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	if wantsJSON(r) {
		return gateway.JsonMarshal(ctx, v)
	}
	return marshalPB(ctx, v)
}

func marshalPB(ctx context.Context, v any) ([]byte, string, error) {
	if err, ok := v.(error); ok {
		if _, isMsg := v.(proto.Message); !isMsg {
			v = mix.ErrRespFrom(err)
		}
	}
	return gateway.ProtobufMarshal(ctx, v)
}

func handleResponse(w http.ResponseWriter, r *http.Request, message proto.Message) error {
	var contentType string
	var buf []byte
	var err error
	switch rb := message.(type) {
	case http.Handler:
		rb.ServeHTTP(w, r)
		return nil
	case mix.Responder:
		rb.Respond(r.Context(), w)
		return nil
	case mix.ResponseBody:
		buf, contentType = rb.ResponseBody()
	case mix.XXXResponseBody:
		buf, contentType, err = marshalForRequest(r, rb.XXX_ResponseBody())
		if err != nil {
			return err
		}
	default:
		buf, contentType, err = marshalForRequest(r, message)
		if err != nil {
			return err
		}
	}
	w.Header().Set(httpx.HeaderContentType, contentType)
	ow := w
	if uw, ok := w.(httpx.Unwrapper); ok {
		ow = uw.Unwrap()
	}
	if recorder, ok := ow.(httpx.RecordBodyer); ok {
		recorder.RecordBody(buf, message)
	}
	_, err = w.Write(buf)
	return err
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	s := gateway.ErrRespFromError(err)
	delete(r.Header, httpx.HeaderTrailer)
	buf, contentType, _ := marshalForRequest(r, s)
	header := w.Header()
	header.Set(httpx.HeaderContentType, contentType)
	mix.WriteErrHeaders(header, s.Code)
	ow := w
	if uw, ok := w.(httpx.Unwrapper); ok {
		ow = uw.Unwrap()
	}
	if recorder, ok := ow.(httpx.RecordBodyer); ok {
		recorder.RecordBody(buf, s)
	}
	w.WriteHeader(mix.StatusFromErrCode(s.Code))
	if _, werr := w.Write(buf); werr != nil {
		grpclog.Infof("Failed to write response: %v", werr)
	}
}
