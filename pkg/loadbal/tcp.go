package loadbal

import (
	"context"
	"net"
)

type TCPBackend struct {
	Dialer net.Dialer
}

func (b TCPBackend) Dial(ctx context.Context) (net.Conn, error) {
	panic("")
}
