package codec

import (
	"fmt"
	"math"
    "strconv"
)

type FIELDTYPE uint8

const (
	FT_PACK		FIELDTYPE = 0
	FT_CHAR		FIELDTYPE = 1
	FT_NUMBER	FIELDTYPE = 2
	FT_STRING	FIELDTYPE = 3
	FT_ARRAY	FIELDTYPE = 4
	FT_MAP		FIELDTYPE = 5
	FT_STRUCT	FIELDTYPE = 6
	FT_FLOAT	FIELDTYPE = 7
	FT_BYTES	FIELDTYPE = 8
	FT_DATE		FIELDTYPE = 9
)


type  FieldType  struct {
	BaseType	FIELDTYPE
	SubType		[]*FieldType
}

func  NewFieldType(ft FIELDTYPE) *FieldType {
	//return &FieldType { BaseType:ft }
	return &FieldType { ft, []*FieldType{} }
}

//------------------------------------------
type PackData	struct {
	m_value		[]byte
	m_curPos	int
}

func  NewPackData() *PackData {
	return &PackData{[]byte{},0}
}

func (pk *PackData) ResetBuff(val string) {
	pk.m_value = []byte(val)
	pk.m_curPos = 0
}
func (pk *PackData) ResetBytes(val []byte) {
	pk.m_value =  val
	pk.m_curPos = 0
}

func (pk *PackData) PackChar(val string) {
	pk.m_value = append(pk.m_value, []byte(val)[0])
	pk.m_curPos += 1
}

func (pk *PackData) PackByte(val byte) {
	pk.m_value = append(pk.m_value, val)
	pk.m_curPos += 1
}

func (pk *PackData) PackBool(val bool) {
	if val {
		pk.PackByte(1)
	} else {
		pk.PackByte(0)
	}
}

func (pk *PackData) PackUint64(val uint64) {
	if val == 0 {
		pk.PackByte(0)
		return
	}
	for val > 0 {
		var ch byte = byte(val & 0x7f)
		val >>= 7
		if val > 0 {
			ch |= 0x80
		}
		pk.PackByte(ch)
	}
}
func (pk *PackData) PackInt64(val int64) {
	pk.PackUint64(uint64(val))
}
func (pk *PackData) PackInt32(val int32) {
	pk.PackUint64(uint64(val))
}
func (pk *PackData) PackUint32(val uint32) {
	pk.PackUint64(uint64(val))
}
func (pk *PackData) PackInt16(val int16) {
	pk.PackUint64(uint64(val))
}
func (pk *PackData) PackUint16(val uint16) {
	pk.PackUint64(uint64(val))
}
func (pk *PackData) PackUint8(val uint8) {
	pk.PackByte(byte(val))
}

func (pk *PackData) PackUNum(val uint) {
	pk.PackUint64(uint64(val))
}
func (pk *PackData) PackNum(val int) {
	pk.PackUint64(uint64(val))
}

func (pk *PackData) PackFieldType(val FIELDTYPE) {
	pk.PackByte(byte(val))
}

func (pk *PackData) PackFieldNum(val uint8) {
	pk.PackByte(byte(val))
}

func (pk *PackData) PackFloat(val float64) {
	bits := math.Float64bits(val)
	var nt uint64 = uint64(bits)
	pk.PackUint64(nt)
}

func (pk *PackData) PackFloat32(val float32) {
	pk.PackFloat(float64(val))
}

func (pk *PackData) PackBytes(val []byte) {
	if val == nil {
		pk.PackNum(0)
		return
	}
	vlen := len(val)
	pk.PackNum(vlen)
	if(vlen > 0) {
		pk.m_value = append(pk.m_value, val...)
		pk.m_curPos += vlen
	}
}

func (pk *PackData) PackString(val string) {
	vt := []byte(val)
	slen := len(vt)
	pk.PackNum(slen)
	if  slen > 0 {
		pk.m_value = append(pk.m_value, vt...)
		pk.m_curPos += slen
	}
}

func (pk *PackData) Data() []byte {
	//fmt.Println(pk.m_curPos)
	return pk.m_value[0:pk.m_curPos]
}

func (pk *PackData) SurData() []byte {
    return pk.m_value[pk.m_curPos:]
}

func (pk *PackData) UnpackByte() byte {
	ch := pk.m_value[pk.m_curPos]
	pk.m_curPos += 1
	return ch
}

func (pk *PackData) UnpackUint64()  uint64 {
	var exp uint64 = 1
	var val uint64 = 0
	var slen int = len(pk.m_value)
	for pk.m_curPos < slen  {
		var ch byte = pk.UnpackByte()
		if (ch&0x80) == 0 {
			val += uint64(ch) * exp
			return val
		}
		ch &= 0x7f
		val += uint64(ch) * exp
		exp <<= 7
	}
	return 0
}
func (pk *PackData) UnpackInt64() int64 {
	return int64(pk.UnpackUint64())
}
func (pk *PackData) UnpackUint32() uint32 {
	return uint32(pk.UnpackUint64())
}
func (pk *PackData) UnpackInt32() int32 {
	return int32(pk.UnpackUint64())
}
func (pk *PackData) UnpackUint16() uint16 {
	return uint16(pk.UnpackUint64())
}
func (pk *PackData) UnpackInt16() int16 {
	return int16(pk.UnpackUint64())
}
func (pk *PackData) UnpackUint8() uint8 {
    return uint8(pk.UnpackByte())
}
func (pk *PackData) UnpackInt() int {
	return int(pk.UnpackUint64())
}
func (pk *PackData) UnpackUint() uint {
	return uint(pk.UnpackUint64())
}
func (pk *PackData) UnpackNum() int32 {
	return pk.UnpackInt32()
}

func (pk *PackData) UnpackFloat() float64 {
	val := pk.UnpackUint64()
	return math.Float64frombits(val)
}
func (pk *PackData) UnpackFloat32() float32 {
	return float32(pk.UnpackFloat())
}

func (pk *PackData) UnpackFieldType() FIELDTYPE {
	return FIELDTYPE(pk.UnpackByte())
}

func (pk *PackData) UnpackField() *FieldType {
    field := NewFieldType(FT_PACK)
    field.BaseType = pk.UnpackFieldType()
    if field.BaseType == FT_ARRAY {
        field.SubType = append(field.SubType, pk.UnpackField())
    } else if field.BaseType == FT_MAP {
        field.SubType = append(field.SubType, pk.UnpackField())
        field.SubType = append(field.SubType, pk.UnpackField())
    }
    return field
}

func (pk *PackData) UnpackFieldNum() uint8 {
	return  pk.UnpackByte()
}

func (pk *PackData) UnpackString() string {
	var slen int = pk.UnpackInt()
	if slen == 0 {
		return string("")
	}
	str := string(pk.m_value[pk.m_curPos:pk.m_curPos+slen])
	pk.m_curPos += slen
	return str
}

func (pk *PackData) Unpack2String(fieldtype FIELDTYPE) string {
    if fieldtype == FT_STRING {
        return pk.UnpackString()
    }
    if fieldtype == FT_NUMBER {
        val := pk.UnpackInt64()
        return strconv.Itoa(int(val))
    }
    panic("unpack to string type error"+string(fieldtype))
}

func (pk *PackData) Unpack2UNumber(fieldtype FIELDTYPE) uint64 {
    if fieldtype == FT_NUMBER {
        return pk.UnpackUint64()
    }
    if fieldtype == FT_STRING {
        str := pk.UnpackString()
        val,_ := strconv.ParseUint(str,10,64)
        return val
    }
    panic("unpack to number error"+string(fieldtype))
}
func (pk *PackData) Unpack2Number(fieldtype FIELDTYPE) int64 {
    if fieldtype == FT_NUMBER {
        return pk.UnpackInt64()
    }
    if fieldtype == FT_STRING {
        str := pk.UnpackString()
        val,_ := strconv.ParseInt(str,10,64)
        return val
    }
    fmt.Println("unpack to number eror")
    panic("unpack to number error"+string(fieldtype))
}

func (pk *PackData) UnpackBytes() []byte {
	var slen int = pk.UnpackInt()
	if slen == 0 {
		return []byte{}
	}
	bt := pk.m_value[pk.m_curPos:pk.m_curPos+slen]
	pk.m_curPos += slen
	return bt
}

func PackLen(buflen int) []byte {
    packer := NewPackData()
    packer.PackNum(buflen)
    return packer.Data()
}

func UnpackLen(buf []byte) (int, []byte) {
    packer := NewPackData()
    packer.ResetBytes(buf)
    rlen := packer.UnpackNum()
    return int(rlen), packer.SurData()
}



