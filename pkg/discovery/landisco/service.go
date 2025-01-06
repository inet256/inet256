package landisco

import (
	"context"
	"encoding/json"
	"time"

	"go.brendoncarroll.net/stdctx/logctx"
	"go.brendoncarroll.net/tai64"
	"go.inet256.org/inet256/pkg/discovery"
	"go.inet256.org/inet256/pkg/inet256"
	"go.inet256.org/inet256/pkg/peers"
	"go.inet256.org/inet256/pkg/serde"
	"golang.org/x/sync/errgroup"
)

var _ discovery.Service = &Service{}

type Service struct {
	Interfaces     []string
	AnnouncePeriod time.Duration
}

func (s *Service) Run(ctx context.Context, params discovery.Params) error {
	b, err := NewBus(ctx, s.Interfaces, s.AnnouncePeriod)
	if err != nil {
		return err
	}
	defer b.Close()

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		<-ctx.Done()
		logctx.Infof(ctx, "closing landisco")
		return b.Close()
	})
	eg.Go(func() error {
		return s.announceLoop(ctx, params, b)
	})
	eg.Go(func() error {
		return s.readLoop(ctx, params, b)
	})
	return eg.Wait()
}

func (s *Service) announceLoop(ctx context.Context, params discovery.Params, b *Bus) error {
	ticker := time.NewTicker(s.AnnouncePeriod)
	defer ticker.Stop()

	now := tai64.Now()
	for {
		if err := b.Announce(ctx, now, params.LocalID, params.GetLocalAddrs()); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			now = tai64.FromGoTime(t)
		}
	}
}

func (s *Service) readLoop(ctx context.Context, params discovery.Params, b *Bus) error {
	msgs := make(chan Message)
	b.Subscribe(msgs)
	defer b.Unsubscribe(msgs)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			if _, err := s.handleMessage(ctx, params, msg); err != nil {
				logctx.Warn(ctx, "while handling message")
			}
		}
	}
}

func (s *Service) handleMessage(ctx context.Context, params discovery.Params, msg Message) (bool, error) {
	// ignore self messages
	if _, _, err := msg.Open([]inet256.ID{params.LocalID}); err == nil {
		return false, nil
	}
	ps := params.Peers.List()
	i, data, err := msg.Open(ps)
	if err != nil {
		return false, err
	}
	adv := Advertisement{}
	if err := json.Unmarshal(data, &adv); err != nil {
		return false, err
	}
	addrs, err := serde.ParseAddrs(params.AddrParser, adv.Transports)
	if err != nil {
		return false, err
	}
	peers.SetAddrs[discovery.TransportAddr](params.Peers, ps[i], addrs)
	return adv.Solicit, nil
}
