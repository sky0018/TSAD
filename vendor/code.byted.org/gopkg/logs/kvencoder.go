package logs

import (
	"bytes"
	"fmt"
	"strconv"
	"io"
)

// KVEncoder .
type KVEncoder interface {
	io.Writer
	Reset()
	AppendKVs(kvs ...interface{})
	EndRecord()
	Bytes() []byte
	String() string
}

// TTLogKVEncoder .
// Key(VLen)=Val
//  Name(3)=zyj
type TTLogKVEncoder struct {
	buf *bytes.Buffer
}

// NewTTLogKVEncoder .
func NewTTLogKVEncoder() *TTLogKVEncoder {
	return &TTLogKVEncoder{
		buf: new(bytes.Buffer),
	}
}

func (tte *TTLogKVEncoder) Write(p []byte) (n int, err error) {
	return tte.buf.Write(p)
}

// Reset .
func (tte *TTLogKVEncoder) Reset() {
	tte.buf.Reset()
}

// Bytes .
func (tte *TTLogKVEncoder) Bytes() []byte {
	return tte.buf.Bytes()
}

// String .
func (tte *TTLogKVEncoder) String() string {
	return tte.buf.String()
}

// AppendKVs .
func (tte *TTLogKVEncoder) AppendKVs(kvs ...interface{}) {
	for i := 0; i+1 < len(kvs); i += 2 {
		k := kvs[i]
		v := kvs[i+1]
		kbytes := []byte(value2Bytes(k))
		vbytes := []byte(value2Bytes(v))
		tte.buf.Write(kbytes)
		tte.buf.Write(equalBytes)
		tte.buf.Write(vbytes)
		tte.buf.Write(spaceBytes)
	}
}

// EndRecord .
func (tte *TTLogKVEncoder) EndRecord() {
	tte.buf.Write(newlineBytes)
}

func value2Bytes(v interface{}) string {
	switch tv := v.(type) {
	case nil:
		return ""
	case string:
		return tv
	case []byte:
		return string(tv)
	case fmt.Stringer:
		if tv == nil {
			return ""
		}
		return tv.String()
	case error:
		if tv == nil {
			return ""
		}
		return tv.Error()
	case int:
		return strconv.Itoa(tv)
	case int16:
		return strconv.FormatInt(int64(tv), 10)
	case int32:
		return strconv.FormatInt(int64(tv), 10)
	case int64:
		return strconv.FormatInt(int64(tv), 10)
	case uint:
		return strconv.FormatUint(uint64(tv), 10)
	case uint16:
		return strconv.FormatUint(uint64(tv), 10)
	case uint32:
		return strconv.FormatUint(uint64(tv), 10)
	case uint64:
		return strconv.FormatUint(uint64(tv), 10)
	case float32:
		return strconv.FormatFloat(float64(tv), 'f', 3, 32)
	case float64:
		return strconv.FormatFloat(float64(tv), 'f', 3, 32)
	default:
		return fmt.Sprint(v)
	}
}

var (
	lBracketBytes = []byte("(")
	rBracketBytes = []byte(")")
	equalBytes    = []byte("=")
	nilBytes      = []byte("nil")
	newlineBytes  = []byte("\n")
)
