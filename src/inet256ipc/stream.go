package inet256ipc

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"go.brendoncarroll.net/p2p"
)

const maxFrameLen = MaxMessageLen + 4

var _ SendReceiver = &StreamFramer{}

type StreamFramer struct {
	rmu sync.Mutex
	br  *bufio.Reader

	wmu sync.Mutex
	w   io.Writer

	bufPool *bufPool
}

func NewStreamFramer(r io.Reader, w io.Writer) *StreamFramer {
	var br *bufio.Reader
	if br2, ok := r.(*bufio.Reader); ok {
		br = br2
	} else {
		br = bufio.NewReaderSize(r, maxFrameLen)
	}
	return &StreamFramer{
		w:       w,
		br:      br,
		bufPool: newBufPool(),
	}
}

func (sf *StreamFramer) Send(ctx context.Context, data []byte) error {
	if len(data) > MaxMessageLen {
		return fmt.Errorf("data too long for StreamFramer len=%d", len(data))
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))

	sf.wmu.Lock()
	defer sf.wmu.Unlock()
	v := p2p.IOVec{
		lenBuf[:],
		data,
	}
	n, err := v.WriteTo(sf.w)
	if err != nil {
		return err
	}
	if n != int64(len(data)+4) {
		return io.ErrShortWrite
	}
	return nil
}

func (sf *StreamFramer) Receive(ctx context.Context, fn func([]byte)) error {
	buf := sf.bufPool.Acquire()
	defer sf.bufPool.Release(buf)
	n, err := func() (int, error) {
		sf.rmu.Lock()
		defer sf.rmu.Unlock()
		var lbuf [4]byte
		if _, err := io.ReadFull(sf.br, lbuf[:]); err != nil {
			return 0, err
		}
		l := binary.BigEndian.Uint32(lbuf[:])
		if l > MaxMessageLen {
			return 0, fmt.Errorf("max message length exceeded %d", l)
		}
		n := int(l)
		if _, err := io.ReadFull(sf.br, buf[:n]); err != nil {
			return 0, err
		}
		return n, nil
	}()
	if err != nil {
		return err
	}
	fn(buf[:n])
	return nil
}

type bufPool struct {
	pool sync.Pool
}

func newBufPool() *bufPool {
	return &bufPool{
		pool: sync.Pool{
			New: func() any {
				return new([maxFrameLen]byte)
			},
		},
	}
}

func (p *bufPool) Acquire() *[maxFrameLen]byte {
	return p.pool.Get().(*[maxFrameLen]byte)
}

func (p *bufPool) Release(buf *[maxFrameLen]byte) {
	p.pool.Put(buf)
}
