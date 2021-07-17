package inet256client

import (
	"net"

	"github.com/inet256/inet256/pkg/inet256"
)

func NewPacketConn(n inet256.Network) net.PacketConn {
	return inet256.NewPacketConn(n)
}
