package inet256client

import (
	"net"
	"testing"

	"github.com/inet256/inet256/networks/floodnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256test"
)

// NewPacketConn wraps the Network n in an adapter exposing the net.PacketConn interface instead.
func NewPacketConn(n inet256.Network) net.PacketConn {
	return inet256.NewPacketConn(n)
}

// NewTestService can be used to spawn an inet256 service without any peering for use in tests
func NewTestService(t testing.TB) inet256.Service {
	return inet256test.NewTestServer(t, floodnet.Factory)
}
