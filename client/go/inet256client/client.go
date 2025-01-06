package inet256client

import (
	"net"
	"os"
	"testing"

	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/inet256http"
	"go.inet256.org/inet256/pkg/inet256mem"
)

const DefaultAPIEndpoint = "unix:///run/inet256.sock"

type (
	Addr = inet256.Addr
	ID   = inet256.ID
)

// NewClient creates an INET256 service using the specified endpoint for the API.
func NewClient(endpoint string) (inet256.Service, error) {
	return inet256http.NewClient(endpoint)
}

// NewEnvClient creates an INET256 service using the environment variables to find the API.
// If you are looking for a inet256.Service constructor, this is probably the one you want.
// It checks the environment variable `INET256_API`
func NewEnvClient() (inet256.Service, error) {
	return NewClient(GetAPIEndpointFromEnv())
}

// NewPacketConn wraps the Node n in an adapter exposing the net.PacketConn interface instead.
func NewPacketConn(n inet256.Node) net.PacketConn {
	return inet256.NewPacketConn(n)
}

// NewTestService can be used to spawn an inet256 service without any peering for use in tests
func NewTestService(t testing.TB) inet256.Service {
	return inet256mem.New()
}

// GetAPIEndpointFromEnv looks for an API endpoint in the environment variable INET256_API.
// If no such variable exists it falls back to DefaultAPIEndpoint
func GetAPIEndpointFromEnv() string {
	endpoint, yes := os.LookupEnv("INET256_API")
	if !yes {
		endpoint = DefaultAPIEndpoint
	}
	return endpoint
}
