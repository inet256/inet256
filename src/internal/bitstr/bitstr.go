package bitstr

import (
	"fmt"
	"strings"
)

// Source is a source of bits.
type Source interface {
	At(i int) bool
	Len() int
}

// BytesLSB is a source of bits, backed by a []byte
// The LSB is considered the first bit in a byte.
type BytesLSB struct {
	Bytes []byte
	Begin int
	End   int
}

func (x BytesLSB) At(i int) bool {
	i += x.Begin
	return x.Bytes[i/8]&mask8LSB(i) > 0
}

func (x BytesLSB) Len() int {
	end := x.End
	if end == 0 {
		end = len(x.Bytes) * 8
	}
	return end - x.Begin
}

func mask8LSB(i int) uint8 {
	return 1 << (i % 8)
}

// BytesMSB is a source of bits, backed by a []byte
// The MSB is considered the first bit in a byte.
type BytesMSB struct {
	Bytes []byte
	Begin int
	End   int
}

func (x BytesMSB) At(i int) bool {
	i += x.Begin
	return x.Bytes[i/8]&mask8MSB(i) > 0
}

func (x BytesMSB) Len() int {
	end := x.End
	if end == 0 {
		end = len(x.Bytes) * 8
	}
	return end - x.Begin
}

func mask8MSB(i int) uint8 {
	return 128 >> (i % 8)
}

type Slice struct {
	Source Source
	Begin  int
	End    int
}

func (x Slice) At(i int) bool {
	return x.Source.At(i + x.Begin)
}

func (x Slice) Len() int {
	return x.End - x.Begin
}

// FromSource creates a new String from a Source
func FromSource(s Source) String {
	buf := Buffer{}
	buf.AppendAll(s)
	return buf.BitString()
}

// Concat returns the result of a concatenated with b
func Concat(a, b String) String {
	buf := Buffer{}
	buf.AppendAll(a)
	buf.AppendAll(b)
	return buf.BitString()
}

// HasPrefix returns true if the first len(p) bits of x are equal to p.
func HasPrefix(x, p Source) bool {
	if x.Len() < p.Len() {
		return false
	}
	for i := 0; i < p.Len(); i++ {
		if x.At(i) != p.At(i) {
			return false
		}
	}
	return true
}

// String is an immutable string of bits
type String struct {
	x string
	l int
}

func (s String) Len() int {
	return s.l
}

func (s String) At(i int) bool {
	return s.x[i/8]&mask8LSB(i) > 0
}

func (s String) Slice(begin, end int) String {
	buf := Buffer{}
	for i := begin; i < end; i++ {
		buf.Append(s.At(i))
	}
	return buf.BitString()
}

func (s String) AsBytesLSB() ([]byte, int) {
	return asBytesLSB(s)
}

func (s String) AsBytesMSB() ([]byte, int) {
	return asBytesMSB(s)
}

func (s String) String() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("BitString{len=%d ", s.l))
	for i := 0; i <= s.Len()/8; i++ {
		if i != 0 {
			sb.WriteString(" ")
		}
		for j := 0; j < 8; j++ {
			index := i*8 + j
			if index >= s.Len() {
				break
			}
			if s.At(index) {
				sb.WriteString("1")
			} else {
				sb.WriteString("0")
			}
		}
	}
	sb.WriteString("}")
	return sb.String()
}

func divCeil(x, z int) int {
	q := x / z
	if x%z > 0 {
		q++
	}
	return q
}

func asBytesLSB(x Source) ([]byte, int) {
	buf := make([]byte, divCeil(x.Len(), 8))
	for i := 0; i < x.Len(); i++ {
		if x.At(i) {
			buf[i/8] |= mask8LSB(i)
		}
	}
	return buf, x.Len()
}

func asBytesMSB(x Source) ([]byte, int) {
	buf := make([]byte, divCeil(x.Len(), 8))
	for i := 0; i < x.Len(); i++ {
		if x.At(i) {
			buf[i/8] |= mask8MSB(i)
		}
	}
	return buf, x.Len()
}
