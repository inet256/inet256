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
	find := func(ctx context.Context, x inet256.Addr) ([]string, error) {
		return client.Find(ctx, p2p.PeerID(x))
	}
	announce := func(ctx context.Context, x inet256.Addr, addrs []string) error {
		return client.Announce(ctx, p2p.PeerID(x), addrs, ttl)
	}
	return discovery.NewPolling(period, find, announce), nil
}
