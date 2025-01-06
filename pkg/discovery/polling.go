package discovery

import (
	"context"
	"time"

	"go.brendoncarroll.net/stdctx/logctx"
	"golang.org/x/sync/errgroup"

	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/peers"
	"go.inet256.org/inet256/pkg/serde"
)

type LookupFunc = func(ctx context.Context, x inet256.Addr) ([]string, error)

type AnnounceFunc = func(ctx context.Context, privateKey inet256.PrivateKey, addrs []string, ttl time.Duration) error

type PollingDiscovery struct {
	Period   time.Duration
	Announce AnnounceFunc
	Lookup   LookupFunc
}

func (s *PollingDiscovery) Run(ctx context.Context, params Params) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.lookupLoop(ctx, params)
	})
	eg.Go(func() error {
		return s.announceLoop(ctx, params)
	})
	return eg.Wait()
}

func (s *PollingDiscovery) lookupLoop(ctx context.Context, params Params) error {
	return s.poll(ctx, func() error {
		for _, target := range params.Peers.List() {
			addrStrs, err := s.Lookup(ctx, target)
			if err != nil {
				return err
			}
			addrs, err := serde.ParseAddrs(params.AddrParser, addrStrs)
			if err != nil {
				return err
			}
			peers.SetAddrs[TransportAddr](params.Peers, target, addrs)
		}
		return nil
	})
}

func (s *PollingDiscovery) announceLoop(ctx context.Context, params Params) error {
	return s.poll(ctx, func() error {
		return s.Announce(ctx, params.PrivateKey, serde.MarshalAddrs(params.GetLocalAddrs()), s.Period*3/2)
	})
}

func (s *PollingDiscovery) poll(ctx context.Context, fn func() error) error {
	ticker := time.NewTicker(s.Period)
	defer ticker.Stop()
	for {
		if err := fn(); err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			logctx.Errorln(ctx, "while polling", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
