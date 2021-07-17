package inet256srv

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

// netAdapter converts an inet256.Network into a p2p.Swarm
type netAdapter struct {
	publicKey p2p.PublicKey
	network   Network
}

func SwarmFromNetwork(network Network, publicKey p2p.PublicKey) p2p.SecureSwarm {
	return &netAdapter{
		publicKey: publicKey,
		network:   network,
	}
}

func (s *netAdapter) Tell(ctx context.Context, dst p2p.Addr, v p2p.IOVec) error {
	return s.network.Tell(ctx, dst.(inet256.Addr), p2p.VecBytes(nil, v))
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
	return []p2p.Addr{inet256.NewAddr(s.publicKey)}
}

func (s *netAdapter) PublicKey() p2p.PublicKey {
	return s.publicKey
}

func (s *netAdapter) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	return s.network.LookupPublicKey(ctx, target.(inet256.Addr))
}

func (s *netAdapter) MTU(ctx context.Context, target p2p.Addr) int {
	return s.network.MTU(ctx, target.(inet256.Addr))
}

func (s *netAdapter) Close() error {
	return s.network.Close()
}

func (s *netAdapter) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (s *netAdapter) ParseAddr(data []byte) (p2p.Addr, error) {
	id := inet256.Addr{}
	if err := id.UnmarshalText(data); err != nil {
		return nil, err
	}
	return id, nil
}

type FindAddrFunc = func(ctx context.Context, prefix []byte, nbits int) (Addr, error)

type BootstrapFunc = func(ctx context.Context) error

var _ Network = &swarmAdapter{}

type swarmAdapter struct {
	swarm         p2p.SecureSwarm
	findAddr      FindAddrFunc
	bootstrapFunc BootstrapFunc
	tells         *TellHub
}

func networkFromSwarm(x p2p.SecureSwarm, findAddr FindAddrFunc, bootstrapFunc BootstrapFunc) Network {
	sa := &swarmAdapter{
		swarm:         x,
		findAddr:      findAddr,
		bootstrapFunc: bootstrapFunc,
		tells:         NewTellHub(),
	}
	ctx := context.Background()
	go func() error {
		buf := make([]byte, TransportMTU)
		for {
			var src, dst p2p.Addr
			n, err := sa.swarm.Recv(ctx, &src, &dst, buf)
			if err != nil {
				return err
			}
			if err := sa.tells.Deliver(ctx, Message{
				Src:     src.(inet256.Addr),
				Dst:     dst.(inet256.Addr),
				Payload: buf[:n],
			}); err != nil {
				return err
			}
		}
	}()
	return sa
}

func (n *swarmAdapter) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.swarm.Tell(ctx, dst, p2p.IOVec{data})
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
	return n.swarm.LookupPublicKey(ctx, target)
}

func (n *swarmAdapter) LocalAddr() Addr {
	return n.swarm.LocalAddrs()[0].(Addr)
}

func (n *swarmAdapter) Bootstrap(ctx context.Context) error {
	return n.bootstrapFunc(ctx)
}

func (n *swarmAdapter) Close() error {
	return n.swarm.Close()
}

func (n *swarmAdapter) MTU(ctx context.Context, target Addr) int {
	return n.swarm.MTU(ctx, target)
}
