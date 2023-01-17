package loadbal

import (
	"context"
	"net"

	"github.com/inet256/go-utp"

	"github.com/inet256/inet256/pkg/inet256"
)

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
