package protocol

import "sync"

var msgPool = sync.Pool{
	New: func() interface{} {
		return &Message{
			Header: &Header{},
		}
	},
}

// GetPooledMsg gets a pooled message.
func GetPooledMsg() *Message {
	return msgPool.Get().(*Message)
}

// FreeMsg puts a msg into the pool.
func FreeMsg(msg *Message) {
	if msg != nil && cap(msg.data) < 1024 {
		msg.Reset()
		msgPool.Put(msg)
	}
}
