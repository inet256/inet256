package inet256client

import (
	"net"
	"os"
	"testing"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256http"
	"github.com/inet256/inet256/pkg/inet256mem"
	"github.com/inet256/inet256/pkg/p2padapter"
)

const DefaultAPIEndpoint = "http://127.0.0.1:2560/nodes/"

type (
	Addr = inet256.Addr
	ID   = inet256.ID
)

// NewClient creates an INET256 service using the specified endpoint for the API.
func NewClient(endpoint string) (inet256.Service, error) {
	return inet256http.NewClient(endpoint), nil
}

// NewEnvClient creates an INET256 service using the environment variables to find the API.
// If you are looking for a inet256.Service constructor, this is probably the one you want.
// It checks the environment variable `INET256_API`
func NewEnvClient() (inet256.Service, error) {
	endpoint, yes := os.LookupEnv("INET256_API")
	if !yes {
		endpoint = DefaultAPIEndpoint
	}
	return NewClient(endpoint)
}

// NewPacketConn wraps the Node n in an adapter exposing the net.PacketConn interface instead.
func NewPacketConn(n inet256.Node) net.PacketConn {
	return inet256.NewPacketConn(n)
}

// NewTestService can be used to spawn an inet256 service without any peering for use in tests
func NewTestService(t testing.TB) inet256.Service {
	return inet256mem.New()
}

// NewSwarm creates a p2p.SecureSwarm from an inet256.Node.
func NewSwarm(node inet256.Node) p2p.SecureSwarm[inet256.Addr] {
	return p2padapter.SwarmFromNode(node)
}
