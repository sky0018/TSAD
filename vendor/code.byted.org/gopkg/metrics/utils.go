package metrics

import (
	"math"
	"reflect"
	"time"
	"unsafe"
)

func toFloat64(v interface{}) (f float64, err error) {
	switch val := v.(type) {
	case float32:
		f = float64(val)
	case float64:
		f = val
	case int:
		f = float64(val)
	case int8:
		f = float64(val)
	case int16:
		f = float64(val)
	case int32:
		f = float64(val)
	case int64:
		f = float64(val)
	case uint8:
		f = float64(val)
	case uint16:
		f = float64(val)
	case uint32:
		f = float64(val)
	case uint64:
		f = float64(val)
	case time.Duration:
		f = float64(val)
	default:
		err = ErrUnKnowValue
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		err = ErrUnKnowValue
	}
	return
}

func appendInt64(b []byte, n int64) []byte {
	if n == 0 {
		return append(b, '0')
	}
	if n < 0 {
		b = append(b, '-')
		n = -n
	}
	var tmp [32]byte
	buf := tmp[:]
	i := len(buf)
	for q := int64(0); n >= 10; {
		i--
		q = n / 10
		buf[i] = '0' + byte(n-q*10)
		n = q
	}
	i--
	buf[i] = '0' + byte(n)
	return append(b, buf[i:]...)
}

func appendFloat64(b []byte, f float64) []byte {
	// apend int part
	n := int64(f)
	if n == math.MinInt64 || n == math.MaxInt64 {
		return append(b, '0')
	}
	if f < 0 || n < 0 {
		b = append(b, '-')
		n = -n
		f = -f
	}
	b = appendInt64(b, n)

	// append float part
	n = int64(f * 100000) // with 5 prec
	if n == math.MaxInt64 || n <= 0 {
		return b
	}
	n = n % 100000
	if n == 0 {
		return b
	}
	j := 5
	for n%10 == 0 {
		n = n / 10
		j--
	}
	b = append(b, '.')
	var tmp [32]byte
	buf := tmp[:]
	buf = appendInt64(buf[:0], n)
	for i := 0; i < j-len(buf); i++ {
		b = append(b, '0')
	}
	return append(b, buf...)
}

func floatStrSize(f float64) int {
	i := 0
	n := int64(f)
	if n == math.MinInt64 || n == math.MaxInt64 {
		return 1
	}
	if n < 0 || f < 0 {
		i++
		n = -n
		f = -f
	}
	if n == 0 {
		i++
	}
	for n > 0 {
		i++
		n = n / 10
	}
	n = int64(f * 100000) // with 5 prec
	if n == math.MaxInt64 || n <= 0 {
		return i
	}
	n = n % 100000 // 99999
	if n == 0 {
		return i
	}
	j := 5
	for n%10 == 0 {
		n = n / 10
		j--
	}
	i += 1 // '.'
	for n > 0 {
		n = n / 10
		i++
		j--
	}
	i += j // 0.000...
	return i
}

func Bytes(s string) []byte {
	p := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	p.Cap = len(s)
	p.Len = p.Cap
	return *(*[]byte)(unsafe.Pointer(p))
}
