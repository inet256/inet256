package inet256d

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"golang.org/x/sync/errgroup"

	"github.com/inet256/inet256/pkg/discovery"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/mesh256"
	"github.com/inet256/inet256/pkg/peers"
)

const defaultPollingPeriod = 30 * time.Second

func (d *Daemon) runAddrDiscovery(ctx context.Context, privateKey inet256.PrivateKey, ds []discovery.AddrService, localAddrs func() []TransportAddr, ps []PeerStore, addrParser p2p.AddrParser[mesh256.TransportAddr]) {
	eg := errgroup.Group{}
	for i := range ds {
		disc := ds[i]
		params := discovery.AddrDiscoveryParams{
			PrivateKey:    privateKey,
			LocalID:       inet256.NewAddr(privateKey.Public()),
			GetLocalAddrs: localAddrs,
			AddressBook:   ps[i],
			AddrParser:    addrParser,
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

func (d *Daemon) runPeerDiscovery(ctx context.Context, localID inet256.Addr, srvs []discovery.PeerService, peerStores []peers.Store[mesh256.TransportAddr], addrSource discovery.AddrSource) {
	if len(srvs) != len(peerStores) {
		panic("len(Services) != len(PeerStores)")
	}
	eg, ctx := errgroup.WithContext(ctx)
	for i, srv := range srvs {
		i := i
		srv := srv
		params := discovery.PeerDiscoveryParams{
			PrivateKey:    nil, // TODO
			LocalID:       localID,
			GetLocalAddrs: addrSource,

			PeerStore: peerStores[i],
			ParseAddr: d.params.TransportAddrParser,
		}
		eg.Go(func() error {
			discovery.RunPeerDiscovery(ctx, srv, params)
			return nil
		})
	}
	eg.Wait()
}
