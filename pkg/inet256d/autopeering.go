package inet256d

import (
	"context"

	"github.com/inet256/inet256/pkg/autopeering"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/inet256srv"
	"github.com/inet256/inet256/pkg/peers"
	"golang.org/x/sync/errgroup"
)

func (d *Daemon) runAutoPeeringServices(ctx context.Context, localID inet256.Addr, srvs []autopeering.Service, peerStores []peers.Store[inet256srv.TransportAddr], addrSource autopeering.AddrSource) {
	if len(srvs) != len(peerStores) {
		panic("len(Services) != len(PeerStores)")
	}
	eg, ctx := errgroup.WithContext(ctx)
	for i, srv := range srvs {
		i := i
		srv := srv
		params := autopeering.Params{
			LocalAddr:  localID,
			AddrSource: addrSource,
			PeerStore:  peerStores[i],
		}
		eg.Go(func() error {
			autopeering.RunForever(ctx, srv, params)
			return nil
		})
	}
	eg.Wait()
}
