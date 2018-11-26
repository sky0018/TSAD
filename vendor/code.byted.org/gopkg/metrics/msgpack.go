package metrics

const (
	msgpackStringHeaderSize = 3
	msgpackArrayHeaderSize  = 3
)

// https://github.com/msgpack/msgpack/blob/master/spec.md#formats-array
/*
array 16 stores an array whose length is upto (2^16)-1 elements:
+--------+--------+--------+~~~~~~~~~~~~~~~~~+
|  0xdc  |YYYYYYYY|YYYYYYYY|    N objects    |
+--------+--------+--------+~~~~~~~~~~~~~~~~~+
*/
func msgpackAppendArrayHeader(b []byte, n uint16) []byte {
	return append(b, []byte{0xdc, byte(n >> 8), byte(n)}...)
}

// https://github.com/msgpack/msgpack/blob/master/spec.md#formats-str
/*
str 16 stores a byte array whose length is upto (2^16)-1 bytes:
+--------+--------+--------+========+
|  0xda  |ZZZZZZZZ|ZZZZZZZZ|  data  |
+--------+--------+--------+========+
*/
func msgpackAppendStringHeader(b []byte, n uint16) []byte {
	return append(b, []byte{0xda, byte(n >> 8), byte(n)}...)
}

func msgpackAppendString(b []byte, s string) []byte {
	b = msgpackAppendStringHeader(b, uint16(len(s)))
	return append(b, s...)
}

func msgpackStringSize(s string) int {
	return msgpackStringHeaderSize + len(s)
}
