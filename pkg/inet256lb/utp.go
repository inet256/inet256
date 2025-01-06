package inet256lb

import (
	"context"
	"net"

	"github.com/inet256/go-utp"

	"go.inet256.org/inet256/pkg/inet256"
)

// NewUTPFrontend creates a new StreamEndpoint listening for UTP connections on a node.
// Nothing else should use node until the endpoint is closed.
func NewUTPFrontend(node inet256.Node) StreamEndpoint {
	pc := inet256.NewPacketConn(node)
	return &utpEndpoint{
		pc: pc,
		s:  utp.NewSocket(pc),
	}
}

// NewUTPBackend creates a new backend for serving requests.
// Outgoing UTP connections will be created to target, from node.
// Nothing else should use node until the endpoint is closed.
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
