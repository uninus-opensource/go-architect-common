package grpc

import (
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

type Response struct {
	Status   string         `json:"status"`
	Data     interface{}    `json:"data,omitempty"`
	Messages *[]InfoMessage `json:"messages,omitempty"`
}

type InfoMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EmptyMarshaler struct{}

func (m *EmptyMarshaler) Marshal(v interface{}) ([]byte, error) {
	return nil, nil
}

func (m *EmptyMarshaler) Unmarshal(data []byte, v interface{}) error {
	return nil
}

func (m *EmptyMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return new(runtime.DecoderFunc)
}

func (m *EmptyMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return new(runtime.EncoderFunc)
}

func (m *EmptyMarshaler) ContentType() string {
	return ""
}
