/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package gateway

import (
	"context"
	"net/http"
	"net/url"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hopeio/gox/context/httpctx"
	"github.com/hopeio/gox/log"
	httpx "github.com/hopeio/gox/net/http"
	"github.com/hopeio/gox/net/http/grpc/gateway"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type GatewayHandler func(context.Context, *runtime.ServeMux)

func New(opts ...runtime.ServeMuxOption) *runtime.ServeMux {
	opts = append([]runtime.ServeMuxOption{
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &JSONPb{}),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			area, err := url.PathUnescape(req.Header.Get(httpx.HeaderArea))
			if err != nil {
				area = ""
			}
			var token = httpx.GetToken(req)
			return metadata.MD{
				httpx.HeaderArea:          {area},
				httpx.HeaderDeviceInfo:    {req.Header.Get(httpx.HeaderDeviceInfo)},
				httpx.HeaderLocation:      {req.Header.Get(httpx.HeaderLocation)},
				httpx.HeaderAuthorization: {token},
			}
		}),
		runtime.WithIncomingHeaderMatcher(gateway.InComingHeaderMatcher),
		runtime.WithOutgoingHeaderMatcher(gateway.OutgoingHeaderMatcher),
		runtime.WithForwardResponseOption(ForwardResponseMessage),
		runtime.WithRoutingErrorHandler(RoutingErrorHandler),
		runtime.WithErrorHandler(CustomHttpError),
	}, opts...)
	return runtime.NewServeMux(opts...)
}

func ForwardResponseMessage(ctx context.Context, writer http.ResponseWriter, message proto.Message) error {
	var buf []byte
	var err error
	contentType := httpx.ContentTypeJson
	switch rb := message.(type) {
	case http.Handler:
		if ctxx, ok := httpctx.FromContext(ctx); ok {
			rb.ServeHTTP(ctxx.ReqCtx.ResponseWriter, ctxx.ReqCtx.Request)
			return nil
		}
	case httpx.Responder:
		rb.Respond(ctx, writer)
		return nil
	case httpx.ResponseBody:
		buf, contentType = rb.ResponseBody()
	case httpx.XXXResponseBody:
		buf, err = JsonPb.Marshal(rb.XXX_ResponseBody())
	default:
		buf, err = JsonPb.Marshal(message)
	}
	if err != nil {
		log.Infof("Marshal error: %v", err)
		return err
	}
	writer.Header().Set(httpx.HeaderContentType, contentType)

	if _, err = writer.Write(buf); err != nil {
		log.Infof("Failed to write response: %v", err)
	}
	return nil
}
