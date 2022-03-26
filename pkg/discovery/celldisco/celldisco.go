package celldisco

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/d/celltracker"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
)

func New(token string) (discovery.Service, error) {
	const (
		ttl    = time.Minute
		period = 10 * time.Second
	)
	client, err := celltracker.NewClient(token)
	if err != nil {
		return nil, err
	}
	return &discovery.PollingDiscovery{
		Period: period,
		Announce: func(ctx context.Context, privKey inet256.PrivateKey, addrs []string, ttl time.Duration) error {
			addr := inet256.NewAddr(privKey.Public())
			return client.Announce(ctx, p2p.PeerID(addr), addrs, ttl)
		},
		Find: func(ctx context.Context, x inet256.Addr) ([]string, error) {
			return client.Find(ctx, p2p.PeerID(x))
		},
	}, nil
}
