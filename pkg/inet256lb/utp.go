package inet256lb

import (
	"context"
	"net"

	"github.com/inet256/go-utp"

	"github.com/inet256/inet256/pkg/inet256"
)

type UTPFrontend struct {
	node inet256.Node
	pc   net.PacketConn
	s    *utp.Socket
}

func NewUTPFrontend(node inet256.Node) StreamEndpoint {
	pc := inet256.NewPacketConn(node)
	return &UTPFrontend{
		node: node,
		pc:   pc,
		s:    utp.NewSocket(pc),
	}
}

func (fe *UTPFrontend) Open(ctx context.Context) (net.Conn, error) {
	return fe.s.Listen()
}

func (fe *UTPFrontend) Close() error {
	fe.s.Close()
	fe.pc.Close()
	fe.node.Close()
	return nil
}

// TODO: this leaks the socket.
func DialUTP(ctx context.Context, node inet256.Node, dst inet256.Addr) (net.Conn, error) {
	pc := inet256.NewPacketConn(node)
	s := utp.NewSocket(pc)
	return s.DialContext(ctx, dst)
}

func ListenUTP(ctx context.Context, node inet256.Node) (net.Listener, error) {
	pc := inet256.NewPacketConn(node)
	s := utp.NewSocket(pc)
	return s, nil
}
