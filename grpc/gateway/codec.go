package gateway

import (
	httpx "github.com/hopeio/gox/net/http"
	"google.golang.org/protobuf/proto"
)

type Protobuf struct {
}

func (*Protobuf) ContentType(_ interface{}) string {
	return httpx.ContentTypeProtobuf
}

func (j *Protobuf) Marshal(v any) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

func (j *Protobuf) Name() string {
	return "jsonpb"
}

func (j *Protobuf) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

func (j *Protobuf) Delimiter() []byte {
	return []byte("\n")
}
