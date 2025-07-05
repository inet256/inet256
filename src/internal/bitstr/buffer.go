package bitstr

import (
	"fmt"
)

const (
	wordBits     = 64
	bytesPerWord = wordBits / 8
)

func mask64(i int) uint64 {
	return 1 << (i % wordBits)
}

// Buffer is a mutable resizable bit string.
type Buffer struct {
	x []uint64
	l int
}

// Append adds a single bit to the end of buffer
func (b *Buffer) Append(x bool) {
	if len(b.x)*wordBits <= b.l {
		b.x = append(b.x, 0)
	}
	b.set(b.l, x)
	b.l++
}

// AppendAll adds all the bits in x to the end of buffer.
func (b *Buffer) AppendAll(x Source) {
	for i := 0; i < x.Len(); i++ {
		b.Append(x.At(i))
	}
}

// AppendByteLSB appends all the bits in x, with the LSB first.
func (b *Buffer) AppendByteLSB(x byte) {
	for i := 0; i < 8; i++ {
		b.Append((x & mask8LSB(i)) > 0)
	}
}

// AppendByteMSB appends all the bits in x, with the MSB first.
func (b *Buffer) AppendByteMSB(x byte) {
	for i := 0; i < 8; i++ {
		b.Append((x & mask8MSB(i)) > 0)
	}
}

func (b *Buffer) Set(i int, x bool) {
	if i >= b.l {
		panic(fmt.Sprintf("out of range index=%d len=%d", i, b.l))
	}
	b.set(i, x)
}

func (b *Buffer) set(i int, x bool) {
	if x {
		b.x[i/wordBits] |= mask64(i)
	} else {
		b.x[i/wordBits] &= ^mask64(i)
	}
}

func (b *Buffer) At(i int) bool {
	if b.l <= i {
		panic(fmt.Sprintf("index out of range len=%d index=%d", b.l, i))
	}
	return b.at(i)
}

func (b *Buffer) at(i int) bool {
	return b.x[i/wordBits]&mask64(i) > 0
}

func (b *Buffer) Len() int {
	return b.l
}

func (b *Buffer) BitString() String {
	buf, l := b.AsBytesLSB()
	return String{
		x: string(buf),
		l: l,
	}
}

func (b *Buffer) AsBytesLSB() ([]byte, int) {
	return asBytesLSB(b)
}

func (b *Buffer) AsBytesMSB() ([]byte, int) {
	return asBytesMSB(b)
}

func (b *Buffer) Truncate(n int) {
	b.l = n
}

func (b *Buffer) Reset() {
	b.Truncate(0)
}
