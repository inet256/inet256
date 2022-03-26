package discovery

import (
	"context"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type FindFunc = func(ctx context.Context, x inet256.Addr) ([]string, error)

type AnnounceFunc = func(ctx context.Context, privateKey inet256.PrivateKey, addrs []string, ttl time.Duration) error

type PollingDiscovery struct {
	Period   time.Duration
	Announce AnnounceFunc
	Find     FindFunc
}

func (s *PollingDiscovery) Run(ctx context.Context, params Params) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.findLoop(ctx, params)
	})
	eg.Go(func() error {
		return s.announceLoop(ctx, params)
	})
	return eg.Wait()
}

func (s *PollingDiscovery) findLoop(ctx context.Context, params Params) error {
	return s.poll(ctx, params.Logger, func() error {
		for _, target := range params.AddressBook.ListPeers() {
			addrStrs, err := s.Find(ctx, target)
			if err != nil {
				return err
			}
			addrs, err := serde.ParseAddrs(params.AddrParser, addrStrs)
			if err != nil {
				return err
			}
			params.AddressBook.SetAddrs(target, addrs)
		}
		return nil
	})
}

func (s *PollingDiscovery) announceLoop(ctx context.Context, params Params) error {
	return s.poll(ctx, params.Logger, func() error {
		return s.Announce(ctx, params.PrivateKey, serde.MarshalAddrs(params.GetLocalAddrs()), s.Period*3/2)
	})
}

func (s *PollingDiscovery) poll(ctx context.Context, log *logrus.Logger, fn func() error) error {
	ticker := time.NewTicker(s.Period)
	defer ticker.Stop()
	for {
		if err := fn(); err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			log.Error("while polling", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
