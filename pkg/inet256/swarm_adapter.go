package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
)

var _ Network = &swarmAdapter{}

type swarmAdapter struct {
	peerswarm PeerSwarm
	peers     PeerStore
}

func newSwarmAdapter(x PeerSwarm, peers PeerStore) Network {
	return &swarmAdapter{
		peerswarm: x,
		peers:     peers,
	}
}

func (n *swarmAdapter) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.peerswarm.TellPeer(ctx, dst, data)
}

func (n *swarmAdapter) OnRecv(fn RecvFunc) {
	n.peerswarm.OnTell(func(msg *p2p.Message) {
		fn(msg.Src.(p2p.PeerID), msg.Dst.(p2p.PeerID), msg.Payload)
	})
}

func (n *swarmAdapter) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	for _, id := range n.peers.ListPeers() {
		addr := id
		if HasPrefix(addr, prefix, nbits) {
			return addr, nil
		}
	}
	return Addr{}, ErrAddrUnreachable
}

func (n *swarmAdapter) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return n.peerswarm.LookupPublicKey(target), nil
}

func (n *swarmAdapter) Close() error {
	return n.peerswarm.Close()
}

func (n *swarmAdapter) MTU(ctx context.Context, target Addr) int {
	return n.peerswarm.MTU(ctx, target)
}
