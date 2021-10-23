package inet256client

import (
	"net"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
)

// NewPacketConn wraps the Node n in an adapter exposing the net.PacketConn interface instead.
func NewPacketConn(n inet256.Node) net.PacketConn {
	return inet256.NewPacketConn(n)
}

// NewTestService can be used to spawn an inet256 service without any peering for use in tests
func NewTestService(t testing.TB) inet256.Service {
	return inet256srv.NewTestServer(t, beaconnet.Factory)
}

// NewSwarm creates a p2p.SecureSwarm from an inet256.Node.
func NewSwarm(n inet256.Node, pubKey p2p.PublicKey) p2p.SecureSwarm {
	return inet256srv.SwarmFromNode(n)
}
