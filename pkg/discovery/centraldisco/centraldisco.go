package centraldisco

import (
	"time"

	"go.inet256.org/inet256/pkg/discovery"
	"go.inet256.org/inet256/pkg/discovery/centraldisco/internal"
	"google.golang.org/grpc"
)

func NewService(client *Client, period time.Duration) discovery.Service {
	return &discovery.PollingDiscovery{
		Period:   period,
		Lookup:   client.Lookup,
		Announce: client.Announce,
	}
}

func RegisterServer(gs *grpc.Server, s *Server) {
	internal.RegisterDiscoveryServer(gs, s)
}
