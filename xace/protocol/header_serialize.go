package protocol

import (
    "xace/codec"
)

const (
    MaxMethodLen int = 127
    DefaultMaxBodyLen int = 1024*1024*64 - 16
)


// Values returns Metadata.
func (head *Header) PackData() []byte {
    packer := codec.NewPackData()
    var fieldnum uint8 = 5
    for {
        if len(head.Metadata) == 0 {
            fieldnum--
        } else {
            break
        }
        if head.SeqId == 0 {
            fieldnum--
        } else {
            break
        }
        if head.CallType == 2 {
            fieldnum--
        } else {
            break
        }
        break
    }
    packer.PackFieldNum(fieldnum)

    packer.PackFieldType(codec.FT_STRING)
    packer.PackString(head.ServicePath)

    packer.PackFieldType(codec.FT_STRING)
    packer.PackString(head.ServiceMethod)

    if fieldnum == 2 {
        return packer.Data()
    }
    packer.PackFieldType(codec.FT_CHAR)
    packer.PackUint8(uint8(head.CallType))

    if fieldnum == 3 {
        return packer.Data()
    }
    packer.PackFieldType(codec.FT_NUMBER)
    packer.PackUint64(head.SeqId)

    if fieldnum == 4 {
        return packer.Data()
    }
    packer.PackFieldType(codec.FT_MAP)
    packer.PackFieldType(codec.FT_STRING)
    packer.PackFieldType(codec.FT_STRING)
    ml := len(head.Metadata)
    packer.PackUint64(uint64(ml))
    for _k,_v := range head.Metadata {
        packer.PackString(_k)
        packer.PackString(_v)
    }
    return packer.Data()
}

func (head *Header) UnpackData(packer *codec.PackData)  {
	if head.Metadata == nil {
		head.Metadata = map[string]string{}
	}
    fieldnum := packer.UnpackFieldNum()
    if fieldnum < 1 {
        panic("aacehead fieldnum")
    }
    fieldtype := packer.UnpackFieldType()
    head.ServicePath = packer.Unpack2String(fieldtype)

    if fieldnum < 2 {
        return
    }
    fieldtype = packer.UnpackFieldType()
    head.ServiceMethod = packer.Unpack2String(fieldtype)

    if fieldnum < 3 {
        return
    }
    fieldtype = packer.UnpackFieldType()
    if fieldtype != codec.FT_CHAR {
        panic("unpack fieldtype error not char.")
    }
    head.CallType = MessageType(packer.UnpackUint8())

    if fieldnum < 4 {
        return
    }
    fieldtype = packer.UnpackFieldType()
    head.SeqId = packer.Unpack2UNumber(fieldtype)

    if fieldnum < 5 {
        return
    }
    field := packer.UnpackField()
    if field.BaseType != codec.FT_MAP || field.SubType[0].BaseType != codec.FT_STRING || field.SubType[1].BaseType != codec.FT_STRING {
        panic("aacehead unpack reserved fieldtype error")
    }
    rnum := packer.UnpackUint32()
    for  ; rnum > 0 ; rnum-- {
        key := packer.UnpackString()
        val := packer.UnpackString()
        head.Metadata[key] = val
    }
}

