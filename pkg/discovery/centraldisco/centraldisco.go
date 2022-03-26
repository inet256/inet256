package centraldisco

import (
	"time"

	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/discovery/centraldisco/internal"
	"google.golang.org/grpc"
)

func NewService(client *Client, period time.Duration) discovery.Service {
	return &discovery.PollingDiscovery{
		Period:   period,
		Find:     client.Find,
		Announce: client.Announce,
	}
}

func RegisterServer(gs *grpc.Server, s *Server) {
	internal.RegisterDiscoveryServer(gs, s)
}
