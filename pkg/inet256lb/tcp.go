package inet256lb

import (
	"context"
	"net"
	"net/netip"
)

type TCPIPBackend struct {
	Endpoint string
	Dialer   net.Dialer
}

func (b *TCPIPBackend) Dial(ctx context.Context) (net.Conn, error) {
	return b.Dialer.DialContext(ctx, "tcp", b.Endpoint)
}

func ParseTCPIP(x string) (netip.AddrPort, error) {
	return netip.ParseAddrPort(x)
}
