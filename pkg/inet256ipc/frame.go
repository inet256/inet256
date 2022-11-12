package inet256ipc

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

const (
	// MaxFrameLen is the maximum length of a frame on the wire
	MaxFrameLen = 4 + MaxMessageLen
	// MaxFrameBodyLen is the maximum length of the body of a frame.
	MaxFrameBodyLen = MaxMessageLen
)

// Frame is a length-prefixed frame suitable for transmitting messages over a stream.
type Frame []byte

func NewFrame() Frame {
	return make([]byte, MaxFrameLen)
}

func (fr Frame) SetLen(l int) {
	if l > MaxFrameLen {
		panic(l)
	}
	binary.BigEndian.PutUint32(fr[:4], uint32(l))
}

func (fr Frame) Len() int {
	return int(binary.BigEndian.Uint32(fr[:4]))
}

// End is the end of the frame in the underlying slice
func (fr Frame) End() int {
	end := 4 + fr.Len()
	if end > len(fr) {
		return len(fr)
	}
	return end
}

func (fr Frame) Body() []byte {
	return fr[4:fr.End()]
}

// Payload is the part of the frame to be transmitted on the wire
func (fr Frame) Payload() []byte {
	return fr[:fr.End()]
}

type Framer interface {
	WriteFrame(ctx context.Context, fr Frame) error
	ReadFrame(ctx context.Context, fr Frame) error
}

type StreamFramer struct {
	rw  io.ReadWriter
	br  *bufio.Reader
	wmu sync.Mutex
	rmu sync.Mutex
}

func NewStreamFramer(rw io.ReadWriter) *StreamFramer {
	return &StreamFramer{
		rw: rw,
		br: bufio.NewReaderSize(rw, MaxFrameLen),
	}
}

func (sf *StreamFramer) WriteFrame(ctx context.Context, fr Frame) error {
	sf.wmu.Lock()
	defer sf.wmu.Unlock()
	_, err := sf.rw.Write(fr.Payload())
	return err
}

func (sf *StreamFramer) ReadFrame(ctx context.Context, fr Frame) error {
	sf.rmu.Lock()
	defer sf.rmu.Unlock()
	var lbuf [4]byte
	if _, err := io.ReadFull(sf.br, lbuf[:]); err != nil {
		return err
	}
	l := binary.BigEndian.Uint32(lbuf[:])
	if l > MaxFrameLen {
		return fmt.Errorf("max frame length exceeded %d", l)
	}
	fr.SetLen(int(l))
	if _, err := io.ReadFull(sf.br, fr.Body()); err != nil {
		return err
	}
	return nil
}

type framePool struct {
	pool sync.Pool
}

func newFramePool() *framePool {
	return &framePool{
		pool: sync.Pool{
			New: func() any {
				return NewFrame()
			},
		},
	}
}

func (p *framePool) Acquire() Frame {
	return p.pool.Get().(Frame)
}

func (p *framePool) Release(fr Frame) {
	fr.SetLen(0)
	p.pool.Put(fr)
}
