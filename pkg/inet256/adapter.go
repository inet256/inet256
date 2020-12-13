package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
)

var _ peerswarm.Swarm = &netAdapter{}

// netAdapter converts an inet256.Network into a swarm
type netAdapter struct {
	publicKey p2p.PublicKey
	network   Network
}

func SwarmFromNetwork(network Network, publicKey p2p.PublicKey) peerswarm.Swarm {
	return &netAdapter{
		publicKey: publicKey,
		network:   network,
	}
}

func (s *netAdapter) Tell(ctx context.Context, dst p2p.Addr, data []byte) error {
	return s.TellPeer(ctx, dst.(p2p.PeerID), data)
}

func (s *netAdapter) TellPeer(ctx context.Context, dst p2p.PeerID, data []byte) error {
	return s.network.Tell(ctx, dst, data)
}

func (s *netAdapter) OnTell(fn p2p.TellHandler) {
	s.network.OnRecv(func(src, dst Addr, data []byte) {
		fn(&p2p.Message{
			Dst:     dst,
			Src:     src,
			Payload: data,
		})
	})
}

func (s *netAdapter) LocalAddrs() []p2p.Addr {
	return []p2p.Addr{p2p.NewPeerID(s.publicKey)}
}

func (s *netAdapter) PublicKey() p2p.PublicKey {
	return s.publicKey
}

func (s *netAdapter) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	return s.network.LookupPublicKey(ctx, target.(p2p.PeerID))
}

func (s *netAdapter) MTU(ctx context.Context, target p2p.Addr) int {
	return s.network.MTU(ctx, target.(p2p.PeerID))
}

func (s *netAdapter) Close() error {
	return s.network.Close()
}

func (s *netAdapter) ParseAddr(data []byte) (p2p.Addr, error) {
	id := p2p.PeerID{}
	if err := id.UnmarshalText(data); err != nil {
		return nil, err
	}
	return id, nil
}

var _ Network = &swarmAdapter{}

type swarmAdapter struct {
	peerswarm PeerSwarm
	findAddr  func(ctx context.Context, prefix []byte, nbits int) (Addr, error)
}

func networkFromSwarm(x PeerSwarm, findAddr func(ctx context.Context, prefix []byte, nbits int) (Addr, error)) Network {
	return &swarmAdapter{
		peerswarm: x,
		findAddr:  findAddr,
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
	return n.findAddr(ctx, prefix, nbits)
}

func (n *swarmAdapter) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return n.peerswarm.LookupPublicKey(ctx, target)
}

func (n *swarmAdapter) Close() error {
	return n.peerswarm.Close()
}

func (n *swarmAdapter) MTU(ctx context.Context, target Addr) int {
	return n.peerswarm.MTU(ctx, target)
}
