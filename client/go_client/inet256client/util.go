package inet256client

import (
	"net"

	"github.com/inet256/inet256/pkg/inet256"
)

// NewPacketConn wraps the Network n in an adapter exposing the net.PacketConn interface instead.
func NewPacketConn(n inet256.Network) net.PacketConn {
	return inet256.NewPacketConn(n)
}
