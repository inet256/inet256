package inet256srv

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/inet256/inet256/pkg/inet256"
)

var _ peerswarm.Swarm = &netAdapter{}

// netAdapter converts an inet256.Network into a peerswarm.Swarm
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

func (s *netAdapter) Tell(ctx context.Context, dst p2p.Addr, v p2p.IOVec) error {
	return s.TellPeer(ctx, dst.(p2p.PeerID), v)
}

func (s *netAdapter) TellPeer(ctx context.Context, dst p2p.PeerID, v p2p.IOVec) error {
	return s.network.Tell(ctx, dst, p2p.VecBytes(nil, v))
}

func (s *netAdapter) Recv(ctx context.Context, src, dst *p2p.Addr, buf []byte) (int, error) {
	var src2, dst2 Addr
	n, err := s.network.Recv(ctx, &src2, &dst2, buf)
	if err != nil {
		return 0, err
	}
	*src = src2
	*dst = dst2
	return n, nil
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

func (s *netAdapter) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (s *netAdapter) ParseAddr(data []byte) (p2p.Addr, error) {
	id := p2p.PeerID{}
	if err := id.UnmarshalText(data); err != nil {
		return nil, err
	}
	return id, nil
}

type FindAddrFunc = func(ctx context.Context, prefix []byte, nbits int) (Addr, error)

type BootstrapFunc = func(ctx context.Context) error

var _ Network = &swarmAdapter{}

type swarmAdapter struct {
	peerswarm     PeerSwarm
	findAddr      FindAddrFunc
	bootstrapFunc BootstrapFunc
	tells         *TellHub
}

func networkFromSwarm(x PeerSwarm, findAddr FindAddrFunc, bootstrapFunc BootstrapFunc) Network {
	sa := &swarmAdapter{
		peerswarm:     x,
		findAddr:      findAddr,
		bootstrapFunc: bootstrapFunc,
		tells:         NewTellHub(),
	}
	ctx := context.Background()
	go func() error {
		buf := make([]byte, TransportMTU)
		for {
			var src, dst p2p.Addr
			n, err := sa.peerswarm.Recv(ctx, &src, &dst, buf)
			if err != nil {
				return err
			}
			if err := sa.tells.Deliver(ctx, Message{
				Src:     src.(p2p.PeerID),
				Dst:     dst.(p2p.PeerID),
				Payload: buf[:n],
			}); err != nil {
				return err
			}
		}
	}()
	return sa
}

func (n *swarmAdapter) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.peerswarm.TellPeer(ctx, dst, p2p.IOVec{data})
}

func (net *swarmAdapter) Recv(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	return net.tells.Recv(ctx, src, dst, buf)
}

func (net *swarmAdapter) WaitRecv(ctx context.Context) error {
	return net.tells.Wait(ctx)
}

func (n *swarmAdapter) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	return n.findAddr(ctx, prefix, nbits)
}

func (n *swarmAdapter) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return n.peerswarm.LookupPublicKey(ctx, target)
}

func (n *swarmAdapter) LocalAddr() Addr {
	return n.peerswarm.LocalAddrs()[0].(Addr)
}

func (n *swarmAdapter) Bootstrap(ctx context.Context) error {
	return n.bootstrapFunc(ctx)
}

func (n *swarmAdapter) Close() error {
	return n.peerswarm.Close()
}

func (n *swarmAdapter) MTU(ctx context.Context, target Addr) int {
	return n.peerswarm.MTU(ctx, target)
}
