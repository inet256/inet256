package inet256

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type packetConn struct {
	n Node

	mu                          sync.RWMutex
	readDeadline, writeDeadline *time.Time
}

// NewPacketConn wraps a node with the net.PacketConn interface
func NewPacketConn(n Node) net.PacketConn {
	return &packetConn{n: n}
}

func (pc *packetConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	dst, err := convertAddr(addr)
	if err != nil {
		return 0, err
	}
	ctx, cf := pc.getWriteContext()
	defer cf()
	if err = pc.n.Tell(ctx, dst, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (pc *packetConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	ctx, cf := pc.getReadContext()
	defer cf()
	if err = pc.n.Receive(ctx, func(m Message) {
		n = copy(p, m.Payload)
		addr = m.Src
	}); err != nil {
		return n, nil, err
	}
	return n, addr, nil
}

func (pc *packetConn) LocalAddr() net.Addr {
	return pc.n.LocalAddr()
}

func (pc *packetConn) Close() error {
	return pc.n.Close()
}

func (pc *packetConn) SetDeadline(t time.Time) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.readDeadline = &t
	pc.writeDeadline = &t
	return nil
}

func (pc *packetConn) SetWriteDeadline(t time.Time) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.writeDeadline = &t
	return nil
}

func (pc *packetConn) SetReadDeadline(t time.Time) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.readDeadline = &t
	return nil
}

func (pc *packetConn) getReadContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	pc.mu.RLock()
	dl := pc.readDeadline
	pc.mu.RUnlock()
	if dl != nil {
		return context.WithDeadline(ctx, *dl)
	} else {
		return context.WithCancel(ctx)
	}
}

func (pc *packetConn) getWriteContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	pc.mu.RLock()
	dl := pc.writeDeadline
	pc.mu.RUnlock()
	if dl != nil {
		return context.WithDeadline(ctx, *dl)
	} else {
		return context.WithCancel(ctx)
	}
}

func convertAddr(x net.Addr) (Addr, error) {
	y, ok := x.(Addr)
	if !ok {
		return Addr{}, errors.Errorf("invalid address: %v", x)
	}
	return y, nil
}
