package inet256client

import (
	"net"
	"os"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks/floodnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/p2padapter"
)

const defaultAPIAddr = inet256d.DefaultAPIEndpoint

type (
	Addr = inet256.Addr
	ID   = inet256.ID
)

func NewExtendedClient(endpoint string) (mesh256.Service, error) {
	return inet256grpc.NewExtendedClient(endpoint)
}

// NewClient creates an INET256 service using the specified endpoint for the API.
func NewClient(endpoint string) (inet256.Service, error) {
	return NewExtendedClient(endpoint)
}

// NewEnvClient creates an INET256 service using the environment variables to find the API.
// If you are looking for a inet256.Service constructor, this is probably the one you want.
// It checks the environment variable `INET256_API`
func NewEnvClient() (inet256.Service, error) {
	endpoint, yes := os.LookupEnv("INET256_API")
	if !yes {
		endpoint = defaultAPIAddr
	}
	return NewClient(endpoint)
}

// NewPacketConn wraps the Node n in an adapter exposing the net.PacketConn interface instead.
func NewPacketConn(n inet256.Node) net.PacketConn {
	return inet256.NewPacketConn(n)
}

// NewTestService can be used to spawn an inet256 service without any peering for use in tests
func NewTestService(t testing.TB) inet256.Service {
	return mesh256.NewTestServer(t, floodnet.Factory)
}

// NewSwarm creates a p2p.SecureSwarm from an inet256.Node.
func NewSwarm(n inet256.Node) p2p.SecureSwarm[inet256.Addr] {
	return p2padapter.P2PSwarmFromNode(n)
}
