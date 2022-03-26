package centraldisco

import (
	"time"

	"github.com/inet256/inet256/pkg/discovery"
)

func NewService(client *Client) discovery.Service {
	return &discovery.PollingDiscovery{
		Period:   10 * time.Second,
		Find:     client.Find,
		Announce: client.Announce,
	}
}
