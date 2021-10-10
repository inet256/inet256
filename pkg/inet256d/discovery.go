package inet256d

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (d *Daemon) runDiscoveryServices(ctx context.Context, localID inet256.Addr, ds []discovery.Service, localAddrs func() []p2p.Addr, ps []inet256.PeerStore) {
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		params := discovery.Params{
			LocalID:       localID,
			GetLocalAddrs: localAddrs,
			AddressBook:   ps[i],
			Logger:        logrus.StandardLogger(),
		}
		eg.Go(func() error {
			discovery.RunForever(ctx, disc, params)
			return nil
		})
	}
	eg.Wait()
}

func adaptTransportAddrs(f func() ([]p2p.Addr, error)) func() []p2p.Addr {
	return func() []p2p.Addr {
		addrs, err := f()
		if err != nil {
			panic(err)
		}
		return addrs
	}
}
