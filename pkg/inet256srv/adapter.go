package inet256srv

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

// netAdapter converts an inet256.Network into a p2p.Swarm
type netAdapter struct {
	network Network
}

// SwarmFromNetwork converts a Network into a p2p.Swarm
func SwarmFromNetwork(network Network) p2p.SecureSwarm {
	return &netAdapter{
		network: network,
	}
}

func (s *netAdapter) Tell(ctx context.Context, dst p2p.Addr, v p2p.IOVec) error {
	return s.network.Tell(ctx, dst.(inet256.Addr), p2p.VecBytes(nil, v))
}

func (s *netAdapter) Receive(ctx context.Context, fn p2p.TellHandler) error {
	return s.network.Receive(ctx, func(msg inet256.Message) {
		fn(p2p.Message{
			Src:     msg.Src,
			Dst:     msg.Dst,
			Payload: msg.Payload,
		})
	})
}

func (s *netAdapter) LocalAddrs() []p2p.Addr {
	return []p2p.Addr{inet256.NewAddr(s.network.PublicKey())}
}

func (s *netAdapter) PublicKey() p2p.PublicKey {
	return s.network.PublicKey()
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
}

func networkFromSwarm(x p2p.SecureSwarm, findAddr FindAddrFunc, bootstrapFunc BootstrapFunc) Network {
	return &swarmAdapter{
		swarm:         x,
		findAddr:      findAddr,
		bootstrapFunc: bootstrapFunc,
	}
}

func (n *swarmAdapter) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.swarm.Tell(ctx, dst, p2p.IOVec{data})
}

func (net *swarmAdapter) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return net.swarm.Receive(ctx, func(m p2p.Message) {
		fn(inet256.Message{
			Dst:     m.Dst.(inet256.Addr),
			Src:     m.Src.(inet256.Addr),
			Payload: m.Payload,
		})
	})
}

func (n *swarmAdapter) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	return n.findAddr(ctx, prefix, nbits)
}

func (n *swarmAdapter) LookupPublicKey(ctx context.Context, target Addr) (inet256.PublicKey, error) {
	return n.swarm.LookupPublicKey(ctx, target)
}

func (n *swarmAdapter) PublicKey() inet256.PublicKey {
	return n.swarm.PublicKey()
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
