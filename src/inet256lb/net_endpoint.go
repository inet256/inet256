package inet256lb

import (
	"context"
	"net"
	"net/netip"
)

// NewTCPIPBackend creates a StreamEndpoint which dials raddr over tcp to open new connections.
func NewTCPIPBackend(raddr netip.AddrPort) StreamEndpoint {
	return &netDialEndpoint{
		network: "tcp",
		target:  raddr.String(),
	}
}

// NewUNIXStreamBackend creates a StreamEndpoint which dials the unix stream socket at p to open new connections.
func NewUNIXStreamBackend(p string) StreamEndpoint {
	raddr := net.UnixAddr{
		Name: p,
	}
	return &netDialEndpoint{
		network: "unix",
		target:  raddr.String(),
	}
}

// netDialer implements StreamEndpoint
type netDialEndpoint struct {
	network string
	target  string
	dialer  net.Dialer
}

func (e *netDialEndpoint) Open(ctx context.Context) (net.Conn, error) {
	return e.dialer.DialContext(ctx, e.network, e.target)
}

func (e *netDialEndpoint) Close() error {
	return nil
}

func NewUNIXStreamFrontend(p string) (StreamEndpoint, error) {
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: p})
	if err != nil {
		return nil, err
	}
	return netListenEndpoint{l: l}, nil
}

func NewTCPIPFrontend(ap netip.AddrPort) (StreamEndpoint, error) {
	l, err := net.Listen("tcp", ap.String())
	if err != nil {
		return nil, err
	}
	return netListenEndpoint{l: l}, nil
}

var _ StreamEndpoint = netListenEndpoint{}

type netListenEndpoint struct {
	l net.Listener
}

func (e netListenEndpoint) Open(ctx context.Context) (net.Conn, error) {
	return e.l.Accept()
}

func (e netListenEndpoint) Close() error {
	return e.l.Close()
}

func ParseTCPIP(x string) (netip.AddrPort, error) {
	return netip.ParseAddrPort(x)
}
