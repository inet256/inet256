package inet256d

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
)

func (d *Daemon) runDiscovery(ctx context.Context, privateKey inet256.PrivateKey, ds []discovery.Service, localAddrs func() []TransportAddr, ps []PeerStore, addrParser p2p.AddrParser[mesh256.TransportAddr]) {
	autopeering := false
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		if !autopeering {
			ps[i] = discovery.DisableAddRemove(ps[i])
		}
		params := discovery.Params{
			AutoPeering:   autopeering,
			PrivateKey:    privateKey,
			LocalID:       inet256.NewAddr(privateKey.Public()),
			GetLocalAddrs: localAddrs,

			Peers:      ps[i],
			AddrParser: addrParser,
		}
		eg.Go(func() error {
			discovery.RunForever(ctx, disc, params)
			return nil
		})
	}
	eg.Wait()
}

var broadcastTransports = map[string]struct{}{
	mesh256.SecureProtocolName("udp"): {},
}

func adaptTransportAddrs(f func(ctx context.Context) ([]TransportAddr, error)) func() []TransportAddr {
	return func() []TransportAddr {
		addrs, err := f(context.TODO())
		if err != nil {
			panic(err)
		}
		addrs2 := addrs[:0]
		for _, addr := range addrs {
			if _, exists := broadcastTransports[addr.Scheme]; exists {
				addrs2 = append(addrs2, addr)
			}
		}
		return addrs2
	}
}
