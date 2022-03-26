package inet256d

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const defaultPollingPeriod = 30 * time.Second

func (d *Daemon) runDiscoveryServices(ctx context.Context, privateKey inet256.PrivateKey, ds []discovery.Service, localAddrs func() []TransportAddr, ps []PeerStore, addrParser p2p.AddrParser[inet256srv.TransportAddr]) {
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		params := discovery.Params{
			PrivateKey:    privateKey,
			LocalID:       inet256.NewAddr(privateKey.Public()),
			GetLocalAddrs: localAddrs,
			AddressBook:   ps[i],
			AddrParser:    addrParser,
			Logger:        logrus.StandardLogger(),
		}
		eg.Go(func() error {
			discovery.RunForever(ctx, disc, params)
			return nil
		})
	}
	eg.Wait()
}

var broadcastTransports = map[string]struct{}{
	"quic+udp": {},
}

func adaptTransportAddrs(f func() ([]TransportAddr, error)) func() []TransportAddr {
	return func() []TransportAddr {
		addrs, err := f()
		if err != nil {
			panic(err)
		}
		addrs2 := addrs[:0]
		for _, addr := range addrs {
			if _, exists := broadcastTransports[addr.Transport]; exists {
				addrs2 = append(addrs2, addr)
			}
		}
		return addrs2
	}
}
