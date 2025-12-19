package gateway

import (
	jsonx "github.com/hopeio/gox/encoding/json"
	"github.com/hopeio/gox/errors"
	httpx "github.com/hopeio/gox/net/http"
	"google.golang.org/protobuf/proto"
)

type Protobuf struct {
}

func (*Protobuf) ContentType(v interface{}) string {
	if _, ok := v.(proto.Message); ok {
		return httpx.ContentTypeProtobuf
	}
	return httpx.ContentTypeJson
}

func (j *Protobuf) Marshal(v any) ([]byte, error) {
	if p, ok := v.(proto.Message); ok {
		return proto.Marshal(p)
	}
	return jsonx.Marshal(httpx.NewRespData(errors.Success, errors.Success.String(), v))
}

func (j *Protobuf) Name() string {
	return "protobuf"
}

func (j *Protobuf) Unmarshal(data []byte, v interface{}) error {
	if p, ok := v.(proto.Message); ok {
		return proto.Unmarshal(data, p)
	}
	return jsonx.Unmarshal(data, v)
}

func (j *Protobuf) Delimiter() []byte {
	return []byte("\n")
}
