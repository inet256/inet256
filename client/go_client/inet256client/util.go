package inet256client

import (
	"net"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/inet256test"
)

// NewPacketConn wraps the Network n in an adapter exposing the net.PacketConn interface instead.
// inet256.Nodes are also inet256.Networks.  One interface is a superset of the other.
func NewPacketConn(n inet256.Network) net.PacketConn {
	return inet256.NewPacketConn(n)
}

// NewTestService can be used to spawn an inet256 service without any peering for use in tests
func NewTestService(t testing.TB) inet256.Service {
	return inet256test.NewTestServer(t, beaconnet.Factory)
}

// NewSwarm creates a p2p.SecureSwarm from an inet256.Network.
func NewSwarm(n inet256.Network, pubKey p2p.PublicKey) p2p.SecureSwarm {
	return inet256srv.SwarmFromNetwork(n)
}
