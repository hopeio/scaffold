package gateway

import (
	jsonx "github.com/hopeio/gox/encoding/json"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	"google.golang.org/protobuf/proto"
)

func Marshal(accept string, v any) ([]byte, string) {
	if p, ok := v.(proto.Message); ok {
		data, err := proto.Marshal(p)
		if err != nil {
			data = []byte(err.Error())
			return data, httpx.ContentTypeText
		}
		return data, httpx.ContentTypeProtobuf
	}
	data, err := jsonx.Marshal(httpx.NewCommonAnyResp(errors.Success, "", v))
	if err != nil {
		data = []byte(err.Error())
		return data, httpx.ContentTypeText
	}
	return data, httpx.ContentTypeJson
}
