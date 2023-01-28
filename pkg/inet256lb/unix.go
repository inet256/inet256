package inet256lb

import (
	"context"
	"net"
)

type UNIXBackend struct {
	endpoint string
}

func (b *UNIXBackend) Dial(ctx context.Context) (net.Conn, error) {
	raddr := net.UnixAddr{
		Name: b.endpoint,
		Net:  "unix",
	}
	return net.DialUnix("unix", nil, &raddr)
}
