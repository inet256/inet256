package inet256p2p

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
)

var _ peerswarm.Swarm = &netAdapter{}

// netAdapter converts an inet256.Network into a swarm
type netAdapter struct {
	publicKey p2p.PublicKey
	network   inet256.Network
}

func newNetAdapter(network inet256.Network, publicKey p2p.PublicKey) *netAdapter {
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
	s.network.OnRecv(func(src, dst inet256.Addr, data []byte) {
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
