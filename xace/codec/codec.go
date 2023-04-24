package codec

import (
	"bytes"
	//"context"
	"encoding/json"
	//"errors"
	"fmt"
	"reflect"

	//"github.com/tinylib/msgp/msgp"
    //"github.com/vmihailenco/msgpack/v5"
)

// Codec defines the interface that decode/encode payload.
type Codec interface {
	Encode(i interface{}) ([]byte, error)
	Decode(data []byte, i interface{}) error
}

// ByteCodec uses raw slice pf bytes and don't encode/decode.
type ByteCodec struct{}

// Encode returns raw slice of bytes.
func (c ByteCodec) Encode(i interface{}) ([]byte, error) {
	if data, ok := i.([]byte); ok {
		return data, nil
	}
	if data, ok := i.(*[]byte); ok {
		return *data, nil
	}

	return nil, fmt.Errorf("%T is not a []byte", i)
}

// Decode returns raw slice of bytes.
func (c ByteCodec) Decode(data []byte, i interface{}) error {
	reflect.Indirect(reflect.ValueOf(i)).SetBytes(data)
	return nil
}

// JSONCodec uses json marshaler and unmarshaler.
type JSONCodec struct{}

// Encode encodes an object into slice of bytes.
func (c JSONCodec) Encode(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

// Decode decodes an object from slice of bytes.
func (c JSONCodec) Decode(data []byte, i interface{}) error {
	d := json.NewDecoder(bytes.NewBuffer(data))
	d.UseNumber()
	return d.Decode(i)
}

// PBCodec uses protobuf marshaler and unmarshaler.
type AceCodec struct{}

// Encode encodes an object into slice of bytes.
func (c AceCodec) Encode(i interface{}) ([]byte, error) {
    return EncodeArgs(i), nil
}

// Decode decodes an object from slice of bytes.
func (c AceCodec) Decode(data []byte, i interface{}) error {
    return DecodeArgs(data, i)
}

/*
// MsgpackCodec uses messagepack marshaler and unmarshaler.
type MsgpackCodec struct{}

// Encode encodes an object into slice of bytes.
func (c MsgpackCodec) Encode(i interface{}) ([]byte, error) {
	if m, ok := i.(msgp.Marshaler); ok {
		return m.MarshalMsg(nil)
	}
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	// enc.UseJSONTag(true)
	err := enc.Encode(i)
	return buf.Bytes(), err
}

// Decode decodes an object from slice of bytes.
func (c MsgpackCodec) Decode(data []byte, i interface{}) error {
	if m, ok := i.(msgp.Unmarshaler); ok {
		_, err := m.UnmarshalMsg(data)
		return err
	}
	dec := msgpack.NewDecoder(bytes.NewReader(data))
	// dec.UseJSONTag(true)
	err := dec.Decode(i)
	return err
}
*/
