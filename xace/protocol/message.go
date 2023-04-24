package protocol

import (
	//"encoding/binary"
	"errors"
	"fmt"
	"io"

    "xace/util"
    "xace/codec"
    "xace/log"
	//"github.com/valyala/bytebufferpool"
)

var bufferPool = util.NewLimitedPool(512, 4096)

// Compressors are compressors supported by rpcx. You can add customized compressor in Compressors.
var Compressors = map[CompressType]Compressor{
	None: &RawDataCompressor{},
	Gzip: &GzipCompressor{},
}

// MaxMessageLength is the max length of a message.
// Default is 0 that means does not limit length of messages.
// It is used to validate when read messages from io.Reader.
var MaxMessageLength = 0

const (
	magicNumber byte = 0x08
)

func MagicNumber() byte {
	return magicNumber
}

var (
	// ErrMetaKVMissing some keys or values are missing.
	ErrMetaKVMissing = errors.New("wrong metadata lines. some keys or values are missing")
	// ErrMessageTooLong message is too long
	ErrMessageTooLong = errors.New("message is too long")

	ErrUnsupportedCompressor = errors.New("unsupported compressor")
)

const (
	// ServiceError contains error info of service invocation
	ServiceError = "__rpcx_error__"
)

// MessageType is message type of requests and responses.
type MessageType uint8

const (
	// Request is message type of request
	Request MessageType = iota
	// Response is message type of response
	Response

    Notify
)

// MessageStatusType is status of messages.
type MessageStatusType byte

const (
	// Normal is normal requests and responses.
	Normal MessageStatusType = iota
	// Error indicates some errors occur.
	Error
)

// CompressType defines decompression type.
type CompressType byte

const (
	// None does not compress.
	None CompressType = iota
	// Gzip uses gzip compression.
	Gzip
)

// SerializeType defines serialization type of payload.
type SerializeType byte

const (
	// SerializeNone uses raw []byte and don't serialize/deserialize
	SerializeNone SerializeType = iota

	// JSON for payload.
	JSON

	// ProtoBuffer for payload.
    AcePack

    BytePack

	// ProtoBuffer for payload.
	//ProtoBuffer
	// MsgPack for payload
	//MsgPack
	// Thrift
	// Thrift for payload
	//Thrift
)

// Message is the generic type of Request and Response.
type Message struct {
	*Header

	Payload       []byte    // 数据存放
	data          []byte    // 接收缓存

    Retcode     int32
}

// NewMessage creates an empty message.
func NewMessage() *Message {
	return &Message{
		Header: &Header{},
	}
}

func (m *Message) Reset() {
    m.Header = &Header{}
	m.Payload = []byte{}
	m.data = m.data[:0]
    m.Retcode = 0
}

// Header is the first part of Message and has fixed size.
// Format:
//
type Header struct {
    ServicePath     string
	ServiceMethod   string
    CallType        MessageType   //uint8
    Compress        CompressType //byte
    Serialize       SerializeType //byte
    Status          MessageStatusType //byte
    SeqId           uint64
    Metadata        map[string]string
}

func (h *Header) SetData(inter,method string, callType MessageType, seq uint64, params map[string]string) {
    h.ServicePath = inter
    h.ServiceMethod = method
    h.CallType = callType
    h.SeqId = seq
    h.Metadata = params
}

func (h *Header) Get(key string) (string, bool) {
	if len(h.Metadata) == 0 {
		return "", false
	}
	value, ok := h.Metadata[key]
	return value, ok
}

// Set sets key-value pair.
func (h *Header) Set(key string, value string) {
	if len(key) == 0 {
		return
	}
    if  len(value) == 0 {
	    if h.Metadata != nil {
            delete(h.Metadata, key)
        }
        return
    }
	if h.Metadata == nil {
		h.Metadata = map[string]string{}
	}
	h.Metadata[key] = value
}

// MessageType returns the message type.
func (h Header) MessageType() MessageType {
	return  h.CallType
}

// SetMessageType sets message type.
func (h *Header) SetMessageType(mt MessageType) {
	h.CallType = mt
}

// IsHeartbeat returns whether the message is heartbeat message.
func (h Header) IsHeartbeat() bool {
	if h.ServicePath == "AaceCheck" && h.ServiceMethod == "check" {
        return true
    }
    return h.ServicePath == ""
}

// IsOneway returns whether the message is one-way message.
// If true, server won't send responses.
func (h Header) IsOneway() bool {
	return h.CallType == Notify
}

// SetOneway sets the oneway flag.
func (h *Header) SetOneway(oneway bool) {
    h.CallType = Notify
}

// CompressType returns compression type of messages.
func (h Header) CompressType() CompressType {
    return h.Compress
}

// SetCompressType sets the compression type.
func (h *Header) SetCompressType(ct CompressType) {
    h.Compress = ct
}

// MessageStatusType returns the message status type.
func (h Header) MessageStatusType() MessageStatusType {
    return h.Status
}

// SetMessageStatusType sets message status type.
func (h *Header) SetMessageStatusType(mt MessageStatusType) {
    h.Status = mt
}

// SerializeType returns serialization type of payload.
func (h Header) SerializeType() SerializeType {
    return h.Serialize
}

// SetSerializeType sets the serialization type.
func (h *Header) SetSerializeType(st SerializeType) {
    h.Serialize = st
}

// Seq returns sequence number of messages.
func (h Header) Seq() uint64 {
    return h.SeqId
}

// SetSeq sets  sequence number.
func (h *Header) SetSeq(seq uint64) {
    h.SeqId = seq
}

// Clone clones from an message.
func (m Message) Clone() *Message {
	header := *m.Header
	c := GetPooledMsg()
	header.SetCompressType(None)
	c.Header = &header
	c.ServicePath = m.ServicePath
	c.ServiceMethod = m.ServiceMethod
	return c
}

// Encode encodes messages.
func (m Message) Encode() []byte {
	data := m.EncodeSlicePointer()
	return *data
}

// EncodeSlicePointer encodes messages as a byte slice pointer we can use pool to improve.
func (m Message) EncodeSlicePointer() *[]byte {

    headbuf := m.Header.PackData()
    lh := len(headbuf)
    ld := len(m.Payload)
    lbuf := codec.PackLen(lh+ld)
    lf := len(lbuf)

	data := bufferPool.Get(lf+lh+ld)
    copy(*data, lbuf)
	copy((*data)[lf:lf+lh], headbuf)
    copy((*data)[lf+lh:], m.Payload)

	return data
}

// PutData puts the byte slice into pool.
func PutData(data *[]byte) {
	bufferPool.Put(data)
}

// WriteTo writes message to writers.
func (m Message) WriteTo(w io.Writer) (int64, error) {
    data := m.EncodeSlicePointer()
    defer PutData(data)
    nn, err := w.Write(*data)
	n := int64(nn)
	if err != nil {
		return n, err
	}
	return n, err
}

// Read reads a message from r.
func Read(r io.Reader) (*Message, error) {
	msg := NewMessage()
	err := msg.Decode(r)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// Decode decodes a message from reader.
func (m *Message) Decode(r io.Reader) (err error) {
    defer func() {
        if errt := recover(); errt != nil {
            err = fmt.Errorf("decode unpack err. %v", errt)
        }
    }()
    tmpbuf := make([]byte, 8)

	// parse len
	_, err = io.ReadFull(r, tmpbuf[:])
	if err != nil {
		return err
	}
    alen, leftBuf := codec.UnpackLen(tmpbuf)
    if alen <= 0 {
		return fmt.Errorf("decode len error. %d", alen)
    }

    if cap(m.data) >= alen {
        m.data = m.data[0:alen]
    } else {
        m.data = make([]byte, alen)
    }

    llen := len(leftBuf)
    if llen > 0 {
        copy(m.data[0:llen], leftBuf[:])
        alen -= llen
    }

    if alen > 0 {
        _, err = io.ReadFull(r, m.data[llen:])
    }
    if err != nil {
        return err
    }

    packer := codec.NewPackData()
    packer.ResetBytes(m.data)
    m.Header.UnpackData(packer)
    //log.Debugf("unpack head %s:%s %d %d", m.Header.ServicePath, m.Header.ServiceMethod, m.Header.SeqId, m.Header.CallType)
    m.Payload = packer.SurData()
	return err
}

func (m *Message) UnpackRetCode() {
    if len(m.Payload) < 4 {
        log.Warn("unpack retcode error.")
        m.Retcode = -90006
        return
    }
    packer := codec.NewPackData()
    packer.ResetBytes(m.Payload)
    m.Retcode = packer.UnpackInt32()
    m.Payload = packer.SurData()
}



