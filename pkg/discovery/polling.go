package discovery

import (
	"context"
	"time"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/serde"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type FindFunc func(ctx context.Context, x inet256.Addr) ([]string, error)

type AnnounceFunc func(ctx context.Context, x inet256.Addr, addrs []string) error

func NewPolling(period time.Duration, find FindFunc, announce AnnounceFunc) *pollingDiscovery {
	return &pollingDiscovery{
		period:   period,
		find:     find,
		announce: announce,
	}
}

type pollingDiscovery struct {
	period   time.Duration
	find     FindFunc
	announce AnnounceFunc
}

func (s *pollingDiscovery) Run(ctx context.Context, params Params) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return s.findLoop(ctx, params)
	})
	eg.Go(func() error {
		return s.announceLoop(ctx, params)
	})
	return eg.Wait()
}

func (s *pollingDiscovery) findLoop(ctx context.Context, params Params) error {
	return s.poll(ctx, params.Logger, func() error {
		for _, target := range params.AddressBook.ListPeers() {
			addrStrs, err := s.find(ctx, target)
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

func (s *pollingDiscovery) announceLoop(ctx context.Context, params Params) error {
	return s.poll(ctx, params.Logger, func() error {
		return s.announce(ctx, params.LocalID, serde.MarshalAddrs(params.GetLocalAddrs()))
	})
}

func (s *pollingDiscovery) poll(ctx context.Context, log *logrus.Logger, fn func() error) error {
	ticker := time.NewTicker(s.period)
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
