package inet256

import (
	"context"
)

func OneHopFactory(params NetworkParams) Network {
	findAddr := func(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
		for _, id := range params.Peers.ListPeers() {
			if HasPrefix(id[:], prefix, nbits) {
				return id, nil
			}
		}
		return Addr{}, ErrNoAddrWithPrefix
	}
	return networkFromSwarm(params.Swarm, findAddr)
}
