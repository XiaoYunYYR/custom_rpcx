package codec

import (
    "sync"
    "math"
    "fmt"
    "errors"
    "strconv"
    "reflect"
)

type DecBuffer struct {
	data	[]byte
	curpos	int
}

var decbufferPool = sync.Pool{
    New: func() any {
        d:= new(DecBuffer)
        return d
    },
}

func (d *DecBuffer) Reset(buf []byte) {
    d.data = buf
    d.curpos = 0
}

func (d *DecBuffer) UnpackByte() byte {
	ch := d.data[d.curpos]
	d.curpos += 1
	return ch
}

func (d *DecBuffer) UnpackUint64()  uint64 {
	var exp uint64 = 1
	var val uint64 = 0
	var slen int = len(d.data)
	for d.curpos < slen  {
		var ch byte = d.UnpackByte()
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
func (d *DecBuffer) UnpackInt64() int64 {
	return int64(d.UnpackUint64())
}
func (d *DecBuffer) UnpackUint32() uint32 {
	return uint32(d.UnpackUint64())
}
func (d *DecBuffer) UnpackInt32() int32 {
	return int32(d.UnpackUint64())
}
func (d *DecBuffer) UnpackUint16() uint16 {
	return uint16(d.UnpackUint64())
}
func (d *DecBuffer) UnpackInt16() int16 {
	return int16(d.UnpackUint64())
}
func (d *DecBuffer) UnpackUint8() uint8 {
    return uint8(d.UnpackByte())
}
func (d *DecBuffer) UnpackInt() int {
	return int(d.UnpackUint64())
}
func (d *DecBuffer) UnpackUint() uint {
	return uint(d.UnpackUint64())
}

func (d *DecBuffer) UnpackNum() int32 {
	return d.UnpackInt32()
}

func (d *DecBuffer) UnpackFloat() float64 {
	val := d.UnpackUint64()
	return math.Float64frombits(val)
}
func (d *DecBuffer) UnpackFloat32() float32 {
	return float32(d.UnpackFloat())
}

func (d *DecBuffer) UnpackFieldType() FIELDTYPE {
	return FIELDTYPE(d.UnpackByte())
}

func (d *DecBuffer) UnpackField() *FieldType {
    field := NewFieldType(FT_PACK)
    field.BaseType = d.UnpackFieldType()
    if field.BaseType == FT_ARRAY {
        field.SubType = append(field.SubType, d.UnpackField())
    } else if field.BaseType == FT_MAP {
        field.SubType = append(field.SubType, d.UnpackField())
        field.SubType = append(field.SubType, d.UnpackField())
    }
    return field
}

func (d *DecBuffer) UnpackFieldNum() uint8 {
	return  d.UnpackByte()
}

func (d *DecBuffer) UnpackString() string {
	var slen int = d.UnpackInt()
	if slen == 0 {
		return string("")
	}
	str := string(d.data[d.curpos:d.curpos+slen])
	d.curpos += slen
	return str
}

func (d *DecBuffer) Unpack2String(fieldtype FIELDTYPE) string {
    if fieldtype == FT_STRING {
        return d.UnpackString()
    }
    if fieldtype == FT_NUMBER {
        val := d.UnpackInt64()
        return strconv.Itoa(int(val))
    }
    panic("unpack to string type error"+string(fieldtype))
}

func (d *DecBuffer) Unpack2UNumber(fieldtype FIELDTYPE) uint64 {
    if fieldtype == FT_NUMBER {
        return d.UnpackUint64()
    }
    if fieldtype == FT_STRING {
        str := d.UnpackString()
        val,_ := strconv.ParseUint(str,10,64)
        return val
    }
    panic("unpack to number error"+string(fieldtype))
}
func (d *DecBuffer) Unpack2Number(fieldtype FIELDTYPE) int64 {
    if fieldtype == FT_NUMBER {
        return d.UnpackInt64()
    }
    if fieldtype == FT_STRING {
        str := d.UnpackString()
        val,_ := strconv.ParseInt(str,10,64)
        return val
    }
    fmt.Println("unpack to number eror")
    panic("unpack to number error"+string(fieldtype))
}

func (d *DecBuffer) UnpackBytes() []byte {
	var slen int = d.UnpackInt()
	if slen == 0 {
		return []byte{}
	}
	bt := d.data[d.curpos:d.curpos+slen]
	d.curpos += slen
	return bt
}

func (d *DecBuffer) peekField(field *FieldType) {
    switch field.BaseType {
    case FT_CHAR:
        d.curpos += 1
    case FT_NUMBER,FT_FLOAT,FT_DATE:
        d.UnpackUint64()
    case FT_STRING:
        d.UnpackString()
    case FT_BYTES:
        d.UnpackBytes()
    case FT_STRUCT:
        plen := int(d.UnpackFieldNum())
        for i:=0; i < plen; i++ {
            d.PeekField()
        }

    case FT_ARRAY:
        un := int(d.UnpackNum())
        for i:=0; i<un; i++ {
            d.peekField(field.SubType[0])
        }

    case FT_MAP:
        un := int(d.UnpackNum())
        for i:=0; i<un; i++ {
            d.peekField(field.SubType[0])
            d.peekField(field.SubType[1])
        }
    }
}


func (d *DecBuffer) PeekField() {
    field := d.UnpackField()
    d.peekField(field)
}

func allocValue(t reflect.Type) reflect.Value {
    return reflect.New(t).Elem()
}

// value.Kind() != reflect.Ptr  ,  must not ptr 
func (d *DecBuffer) decodeByField(field *FieldType, value reflect.Value) {
    switch field.BaseType {
    case FT_CHAR:
        v := d.UnpackUint8()
        switch value.Kind() {
        case reflect.Bool:
            if v == 0 {
                value.SetBool(false)
            } else {
                value.SetBool(true)
            }
        case reflect.Uint8,reflect.Uint16,reflect.Uint32,reflect.Uint,reflect.Uint64:
            value.SetUint(uint64(v))
        case reflect.Int16,reflect.Int32,reflect.Int64,reflect.Int:
            value.SetInt(int64(v))
        default:
            fmt.Println(value.Kind())
            panic("aace connot transform value type char"+string(value.Kind()))
        }
    case FT_NUMBER:
        v := d.UnpackInt64()
        switch value.Kind() {
        case reflect.Bool:
            if v == 0 {
                value.SetBool(false)
            } else {
                value.SetBool(true)
            }
        case reflect.Uint8,reflect.Uint16,reflect.Uint32,reflect.Uint,reflect.Uint64:
            value.SetUint(uint64(v))

        case reflect.Int16,reflect.Int32,reflect.Int64,reflect.Int:
            value.SetInt(int64(v))
        case reflect.String:
            value.SetString(strconv.Itoa(int(v)))
        default:
            fmt.Println(value.Kind())
            panic("aace cannot transform value type number"+string(value.Kind()))
        }
    case FT_STRING:
        v := d.UnpackString()
        switch value.Kind() {
        case reflect.String:
            value.SetString(v)
        case reflect.Uint32,reflect.Uint,reflect.Uint64,reflect.Int32,reflect.Int64,reflect.Int:
            val,_ := strconv.ParseInt(v,10,64)
            value.SetInt(val)
        default:
            panic("aace cannot transform value type string")
        }

    case FT_BYTES:
        if value.Kind() != reflect.Slice {
            panic("aace cannot transfrom value type bytes")
        }
        v := d.UnpackBytes()
        value.SetBytes(v)

    case FT_ARRAY:
        if value.Kind() != reflect.Slice {
            panic("aace cannot transfrom value type slice")
        }
        un := int(d.UnpackNum())
        if value.Cap() < un {
            value.Set(reflect.MakeSlice(value.Type(), un, un))
        } else {
            value.SetLen(un)
        }

        vt := value.Type().Elem()
        if vt.Kind() == reflect.Ptr {
            for i:=0; i<un; i++ {
                vd := reflect.New(vt.Elem())
                vi := value.Index(i)
                vi.Set(vd)
                d.decodeByField(field.SubType[0],vd.Elem())
            }
        } else {
            for i:=0; i<un; i++ {
                d.decodeByField(field.SubType[0],value.Index(i))
            }
        }

    case FT_MAP:
        if value.Kind() != reflect.Map {
            fmt.Println(value.Kind())
            panic("aace cannot fransfrom value type map")
        }
        un := int(d.UnpackNum())
        if value.IsNil() {
            value.Set(reflect.MakeMapWithSize(value.Type(), un))
        }

        kt := value.Type().Key()    // KT
        vt := value.Type().Elem()   // T / *T

        for i:=0; i<un; i++ {
            km := reflect.New(kt).Elem()
            d.decodeByField(field.SubType[0],km)
            if vt.Kind() == reflect.Ptr {  // vt  *T
                vm := reflect.New(vt.Elem())
                d.decodeByField(field.SubType[1],vm.Elem())
                value.SetMapIndex(km, vm)
            } else {                        // vt T
                vm := reflect.New(vt).Elem()
                d.decodeByField(field.SubType[1],vm)
                value.SetMapIndex(km, vm)
            }
        }

    case FT_STRUCT:
        vt := value.Type()
        unnum := int(d.UnpackFieldNum())
        fieldnum := vt.NumField()
        var i int = 0
        for ; i<unnum && i < fieldnum; i++ {
            fv := value.Field(i)
            if fv.Kind() == reflect.Ptr {
                sf := reflect.New(fv.Type().Elem())  // sf *T
                fv.Set(sf)
                d.decodeValue(fv.Elem())
            } else {
                d.decodeValue(fv)
            }
        }
        for ; i<unnum; i++ {
            d.UnpackField()
        }
    }
}
func (d *DecBuffer) decodeValue(value reflect.Value) {
    field := d.UnpackField()
    d.decodeByField(field, value)
}


func (d *DecBuffer) DecodeValue(arg any) {
    value := reflect.ValueOf(arg)
    if value.Kind() == reflect.Invalid {
        panic("ace: cannot decode invalid value")
    }
    if value.Kind() != reflect.Ptr {
        panic("ace: cannot unpack data 2 value")
    }
    if value.IsNil() {
        panic("ace: cannot unpack nil value")
    }

    //value = reflect.Indirect(value)
    value = value.Elem()

    d.decodeValue(value)
}

func (d *DecBuffer) DecodeStruct(arg any) {
    value := reflect.ValueOf(arg)
    if value.Kind() == reflect.Invalid {
        panic("ace: cannot decode invalid value")
    }
    if value.Kind() != reflect.Ptr {
        panic("ace: cannot unpack data 2 value")
    }
    if value.IsNil() {
        panic("ace: cannot unpack nil value")
    }

    //value = reflect.Indirect(value)
    value = value.Elem()

    if value.Kind() != reflect.Struct {
        panic("ace: cannot unpack data 2 value, only to struct")
    }
    field := NewFieldType(FT_STRUCT)
    d.decodeByField(field, value)
}

func (d *DecBuffer) Decode(args... any) (err error) {
    n := len(args)
    if n < 1 {
        return errors.New("decode to no parameters")
    }
    defer func() {
        if errt := recover(); errt != nil {
            fmt.Println(errt)
            //err = errors.New("decode invalid ")
        }
    }()
    curn := int(d.UnpackFieldNum())
    for i:=0; i<n && i<curn; i++ {
        d.DecodeValue(args[i])
    }
    return err
}

func (d *DecBuffer) DecodeArgs(args any) (err error) {
    defer func() {
        if errt := recover(); errt != nil {
            fmt.Println(errt)
            err = errors.New("decode invalid ")
        }
    }()
    lt, ok := args.([]any)
    if ok {
        n := len(lt)
        if n < 1 {
            return nil
        }
        curn := int(d.UnpackFieldNum())
        for i:=0; i<n && i<curn; i++ {
            d.DecodeValue(lt[i])
        }
        return
    } else {
        d.DecodeStruct(args)
        //err = errors.New("no support decode input type")
    }
    return err
}

func DecodeArgs(buf []byte,args any) error {
    dec := decbufferPool.Get().(*DecBuffer)
    dec.Reset(buf)
    return dec.DecodeArgs(args)
}






