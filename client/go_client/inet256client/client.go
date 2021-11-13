package inet256client

import (
	"net"
	"os"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks/beaconnet"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256d"
	"github.com/inet256/inet256/pkg/inet256grpc"
	"github.com/inet256/inet256/pkg/inet256srv"
)

const defaultAPIAddr = inet256d.DefaultAPIEndpoint

func NewExtendedClient(endpoint string) (inet256srv.Service, error) {
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
	return inet256srv.NewTestServer(t, beaconnet.Factory)
}

// NewSwarm creates a p2p.SecureSwarm from an inet256.Node.
func NewSwarm(n inet256.Node) p2p.SecureSwarm {
	return inet256srv.SwarmFromNode(n)
}
