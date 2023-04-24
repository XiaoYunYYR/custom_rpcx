package codec

import (
    //"io"
    "sync"
    "math"
    "reflect"
)

const tooBig = (1 << 30) << (^uint(0) >> 62)

type EncBuffer struct {
    data    []byte
    scratch [64]byte
}

// global aacehead
var encbufferPool = sync.Pool{
    New: func() any {
        e := new(EncBuffer)
        e.data = e.scratch[0:0]
        return e
    },
}

func encBufferPut(e *EncBuffer) {
    if cap(e.data) > 1024 {
        e.data = e.scratch[0:0]
    } else {
        e.data = e.data[0:0]
    }
}

func (e *EncBuffer) writeByte(c byte) {
    e.data = append(e.data, c)
}

func (e *EncBuffer) Write(p []byte) (int, error) {
    e.data = append(e.data, p...)
    return len(p),nil
}

func (e *EncBuffer) WriteString(s string) {
    e.data = append(e.data, s...)
}

func (e *EncBuffer) Len() int {
    return len(e.data)
}

func (e *EncBuffer) Bytes() []byte {
    return e.data
}

func (e *EncBuffer) Reset() {
    if len(e.data) >= tooBig {
        e.data = e.scratch[0:0]
    } else {
        e.data = e.data[0:0]
    }
}

func (e *EncBuffer) PackChar(val string) {
    e.writeByte([]byte(val)[0])
}

func (e *EncBuffer) PackByte(val byte) {
    e.writeByte(val)
}

func (e *EncBuffer) PackBool(val bool) {
	if val {
        e.writeByte(1)
	} else {
        e.writeByte(0)
	}
}

func (e *EncBuffer) PackUint(val uint64) {
	if val == 0 {
        e.writeByte(0)
		return
	}
	for val > 0 {
		var ch byte = byte(val & 0x7f)
		val >>= 7
		if val > 0 {
			ch |= 0x80
		}
        e.writeByte(ch)
	}
}
func (e *EncBuffer) PackInt(val int64) {
	e.PackUint(uint64(val))
}

func (e *EncBuffer) PackNum(val int) {
	e.PackUint(uint64(val))
}

func (e *EncBuffer) PackFloat(val float64) {
	bits := math.Float64bits(val)
	var nt uint64 = uint64(bits)
	e.PackUint(nt)
}

func (e *EncBuffer) PackBytes(val []byte) {
	if val == nil {
		e.PackNum(0)
		return
	}
	vlen := len(val)
	e.PackNum(vlen)
	if(vlen > 0) {
        e.data = append(e.data, val...)
	}
}

func (e *EncBuffer) PackString(val string) {
	vt := []byte(val)
	slen := len(vt)
	e.PackNum(slen)
	if  slen > 0 {
        e.WriteString(val)
	}
}

func (e *EncBuffer) PackFieldType(val FIELDTYPE) {
	e.writeByte(byte(val))
}

func (e *EncBuffer) PackFieldNum(val int) {
    e.writeByte(byte(val))
}

func (e *EncBuffer) encodeValueNoType(value reflect.Value) {
    if value.Kind() == reflect.Invalid {
        panic("ace: cannot encode nil value")
    }
    if value.Kind() == reflect.Pointer && value.IsNil() {
        panic("ace: cannot encode nil pointer of type: " + value.Type().String())
    }
    value = reflect.Indirect(value)

    switch value.Kind() {
    case reflect.Bool:
        e.PackBool(value.Bool())

    case reflect.Uint8:
        e.PackByte(value.Bytes()[0])

    case reflect.Int,reflect.Int16,reflect.Int32,reflect.Int64:
        e.PackInt(value.Int())

    case reflect.Uint,reflect.Uint16,reflect.Uint32,reflect.Uint64:
        e.PackUint(value.Uint())

    case reflect.Float32, reflect.Float64:
        e.PackFloat(value.Float())

    case reflect.String:
        e.PackString(value.String())

    case reflect.Struct:
        fieldnum := value.NumField()
        e.PackFieldNum(fieldnum)
        for i:=0; i<fieldnum; i++ {
            e.EncodeValue(value.Field(i))
        }
    }
}

func (e *EncBuffer) EncodeValue(value reflect.Value) {
    if value.Kind() == reflect.Invalid {
        panic("ace: cannot encode nil value")
    }
    if value.Kind() == reflect.Pointer && value.IsNil() {
        panic("ace: cannot encode nil pointer of type: " + value.Type().String())
    }
    value = reflect.Indirect(value)

    switch value.Kind() {
    case reflect.Bool:
        e.PackFieldType(FT_CHAR)
        e.PackBool(value.Bool())

    case reflect.Uint8:
        e.PackFieldType(FT_CHAR)
        e.PackByte(value.Bytes()[0])

    case reflect.Int,reflect.Int16,reflect.Int32,reflect.Int64:
        e.PackFieldType(FT_NUMBER)
        e.PackInt(value.Int())

    case reflect.Uint,reflect.Uint16,reflect.Uint32,reflect.Uint64:
        e.PackFieldType(FT_NUMBER)
        e.PackUint(value.Uint())

    case reflect.Float32, reflect.Float64:
        e.PackFieldType(FT_FLOAT)
        e.PackFloat(value.Float())

    case reflect.String:
        e.PackFieldType(FT_STRING)
        e.PackString(value.String())

    case reflect.Struct:
        e.PackFieldType(FT_STRUCT)
        fieldnum := value.NumField()
        e.PackFieldNum(fieldnum)
        for i:=0; i<fieldnum; i++ {
            e.EncodeValue(value.Field(i))
        }

    case reflect.Slice,reflect.Array:
        vt := value.Type()
        vt = vt.Elem()
        if vt.Kind() == reflect.Uint8 {
            e.PackFieldType(FT_BYTES)
            e.PackBytes(value.Bytes())
        } else {
            e.PackFieldType(FT_ARRAY)
            e.PackFieldType(getValFieldType(vt))

            n_ := value.Len()
            e.PackNum(n_)
            for i:=0; i<n_; i++ {
                e.encodeValueNoType(value.Index(i))
            }
        }

    case reflect.Map:
        e.PackFieldType(FT_MAP)
        vt := value.Type()
        kt := vt.Key()
        dt := vt.Elem()
        e.PackFieldType(getValFieldType(kt))
        e.PackFieldType(getValFieldType(dt))

        n_ := value.Len()
        e.PackNum(n_)

        mi := value.MapRange()
        for mi.Next() {
            e.encodeValueNoType(mi.Key())
            e.encodeValueNoType(mi.Value())
        }
    }
}

func (e *EncBuffer) Encode(args... any) {
    n := len(args)
    e.PackFieldNum(n)
    for i:=0; i<n; i++ {
        e.EncodeValue(reflect.ValueOf(args[i]))
    }
}

func (e *EncBuffer) EncodeArgs(args any) {
    lt, ok := args.([]any)
    if ok {
        n := len(lt)
        e.PackFieldNum(n)
        for i:=0; i<n; i++ {
            e.EncodeValue(reflect.ValueOf(lt[i]))
        }
    } else {
        e.PackFieldNum(1)
        e.EncodeValue(reflect.ValueOf(args))
    }
}

func getValFieldType(vt reflect.Type) FIELDTYPE {
    if vt.Kind() == reflect.Ptr {
        vt = vt.Elem()
    }
    switch vt.Kind() {
    case reflect.String:
        return FT_STRING
    case reflect.Int,reflect.Int16,reflect.Int32,reflect.Int64:
        return FT_NUMBER
    case reflect.Uint,reflect.Uint16,reflect.Uint32,reflect.Uint64:
        return FT_NUMBER
    case reflect.Float32,reflect.Float64:
        return FT_FLOAT
    case reflect.Struct:
        return FT_STRUCT
    }
    panic("get value field type error."+vt.Kind().String())
    return FT_PACK
}

func EncodeArgs(args any) []byte {
    enc := encbufferPool.Get().(*EncBuffer)
    defer encBufferPut(enc)
    enc.EncodeArgs(args)
    data := enc.Bytes()
    return data
}

func EncodeRetArgs(retcode int32, data []byte) []byte {
    enc := encbufferPool.Get().(*EncBuffer)
    defer encBufferPut(enc)
    enc.PackUint(uint64(retcode))
    enc.Write(data)
    data2 := enc.Bytes()
    return data2
}

/*
type Encoder struct {
    mutex       sync.Mutex
    w           io.Writer
    byteBuf     EncBuffer
    err         error
}

func NewEncoder(w io.Writer) *Encoder {
    enc := new(Encoder)
    enc.w = w
    return enc
}

func (enc *Encoder) setError(err error) {
    if enc.err == nil {
        enc.err = err
    }
}

func (enc *Encoder) Encode(args... any) error {
    enc.mutex.Lock()
    defer enc.mutex.Unlock()

    enc.err = nil
    enc.byteBuf.Reset()

    // TODO 
    return nil
}
*/
