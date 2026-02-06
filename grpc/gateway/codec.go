package gateway

import (
	"context"

	jsonx "github.com/hopeio/gox/encoding/json"
	httpx "github.com/hopeio/gox/net/http"
	"google.golang.org/protobuf/proto"
)

func ProtobufMarshal(ctx context.Context, v any) ([]byte, string) {
	if p, ok := v.(proto.Message); ok {
		data, err := proto.Marshal(p)
		if err != nil {
			data = []byte(err.Error())
			return data, httpx.ContentTypeText
		}
		return data, httpx.ContentTypeProtobuf
	}
	return JsonMarshal(ctx, v)
}

func JsonMarshal(ctx context.Context, v any) ([]byte, string) {
	data, err := jsonx.Marshal(v)
	if err != nil {
		data = []byte(err.Error())
		return data, httpx.ContentTypeText
	}
	return data, httpx.ContentTypeJson
}
