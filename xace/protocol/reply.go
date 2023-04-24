package protocol

import (
    "sync"
)

type AceReply struct {
    Retcode     int32
    Args        []any
}


func NewAceReply() *AceReply {
    return &AceReply{
        Retcode:    0,
        Args:       make([]any,0,10),
    }
}

var (
    aceReplyPool = sync.Pool{
        New: func() interface{} {
            return NewAceReply()
        },
    }
)

func GetAceReply() *AceReply {
    ar,_ := aceReplyPool.Get().(*AceReply)
    return ar
}

func PutAceReply(a *AceReply) {
    a.Args = a.Args[0:0]
    aceReplyPool.Put(a)
}


