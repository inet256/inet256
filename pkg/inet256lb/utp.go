package inet256lb

import (
	"context"
	"net"

	"github.com/inet256/go-utp"

	"github.com/inet256/inet256/pkg/inet256"
)

func NewUTPFrontend(node inet256.Node) StreamEndpoint {
	pc := inet256.NewPacketConn(node)
	return &utpEndpoint{
		pc: pc,
		s:  utp.NewSocket(pc),
	}
}

func NewUTPBackend(node inet256.Node, target inet256.Addr) StreamEndpoint {
	pc := inet256.NewPacketConn(node)
	return &utpEndpoint{
		pc:     pc,
		s:      utp.NewSocket(pc),
		target: target,
	}
}

type utpEndpoint struct {
	pc     net.PacketConn
	s      *utp.Socket
	target inet256.Addr
}

func (e *utpEndpoint) Open(ctx context.Context) (net.Conn, error) {
	if e.target.IsZero() {
		return e.s.Accept()
	} else {
		return e.s.DialContext(ctx, e.target)
	}
}

func (e *utpEndpoint) Close() error {
	return e.s.Close()
}
