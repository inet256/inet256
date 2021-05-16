package inet256d

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (d *Daemon) runDiscoveryServices(ctx context.Context, localID p2p.PeerID, ds []p2p.DiscoveryService, localAddrs func() []string, ps []inet256.PeerStore) {
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		p := ps[i]
		eg.Go(func() error {
			return d.runDiscoveryService(ctx, localID, disc, localAddrs, p)
		})
	}
	eg.Wait()
}

func (d *Daemon) runDiscoveryService(ctx context.Context, localID p2p.PeerID, ds p2p.DiscoveryService, localAddrs func() []string, ps inet256.PeerStore) error {
	eg := errgroup.Group{}
	// announce loop
	eg.Go(func() error {
		ttl := 60 * time.Second
		ticker := time.NewTicker(60 * time.Second / 2)
		defer ticker.Stop()
		for {
			addrs := localAddrs()
			if err := ds.Announce(ctx, localID, addrs, ttl); err != nil {
				logrus.Error("announce error:", err)
			}
			// TODO: announce, and find, add each to the store
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	})
	// Find loop
	eg.Go(func() error {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			for _, id := range ps.ListPeers() {
				addrs, err := ds.Find(ctx, localID)
				if err != nil {
					logrus.Error("find errored: ", err)
					continue
				}
				ps.SetAddrs(id, addrs)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
			}
		}
	})
	return eg.Wait()
}
