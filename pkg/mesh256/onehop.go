package mesh256

import (
	"context"

	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/p2padapter"
)

// OneHopFactory creates a network which isn't really a network.
// It only supports sending messages to immediate neighbors.
func OneHopFactory(params NetworkParams) Network {
	findAddr := func(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
		for _, id := range params.Peers.ListPeers() {
			if inet256.HasPrefix(id[:], prefix, nbits) {
				return inet256.Addr(id), nil
			}
		}
		return Addr{}, inet256.ErrNoAddrWithPrefix
	}
	waitReady := func(ctx context.Context) error {
		return nil
	}
	return p2padapter.NetworkFromSwarm(params.Swarm, findAddr, waitReady)
}
