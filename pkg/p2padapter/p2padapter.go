package p2padapter

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
)

// netAdapter converts an networks.Network into a p2p.Swarm
type netAdapter struct {
	n networks.Swarm
}

// SwarmFromNetwork converts a Network into a p2p.Swarm
func P2PFromINET256(n networks.Swarm) p2p.SecureSwarm {
	return &netAdapter{
		n: n,
	}
}

func (s *netAdapter) Tell(ctx context.Context, dst p2p.Addr, v p2p.IOVec) error {
	return s.n.Tell(ctx, dst.(inet256.Addr), v)
}

func (s *netAdapter) Receive(ctx context.Context, fn p2p.TellHandler) error {
	return s.n.Receive(ctx, func(msg inet256.Message) {
		fn(p2p.Message{
			Src:     msg.Src,
			Dst:     msg.Dst,
			Payload: msg.Payload,
		})
	})
}

func (s *netAdapter) LocalAddrs() []p2p.Addr {
	return []p2p.Addr{inet256.NewAddr(s.n.PublicKey())}
}

func (s *netAdapter) PublicKey() p2p.PublicKey {
	return s.n.PublicKey()
}

func (s *netAdapter) LookupPublicKey(ctx context.Context, target p2p.Addr) (p2p.PublicKey, error) {
	return s.n.LookupPublicKey(ctx, target.(inet256.Addr))
}

func (s *netAdapter) MTU(ctx context.Context, target p2p.Addr) int {
	return s.n.MTU(ctx, target.(inet256.Addr))
}

func (s *netAdapter) Close() error {
	return s.n.Close()
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

type FindAddrFunc = func(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)

type BootstrapFunc = func(ctx context.Context) error

var _ networks.Network = &swarmAdapter{}

// NetworkFromSwarm creates a networks.Network from a networks.Swarm
func NetworkFromSwarm(x networks.Swarm, findAddr FindAddrFunc, boorstrapFunc BootstrapFunc) networks.Network {
	return &swarmAdapter{
		Swarm:         x,
		findAddr:      findAddr,
		bootstrapFunc: boorstrapFunc,
	}
}

type swarmAdapter struct {
	networks.Swarm
	findAddr      FindAddrFunc
	bootstrapFunc BootstrapFunc
}

func (n *swarmAdapter) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return n.findAddr(ctx, prefix, nbits)
}

func (n *swarmAdapter) Bootstrap(ctx context.Context) error {
	return n.bootstrapFunc(ctx)
}

func (n *swarmAdapter) MTU(ctx context.Context, target inet256.Addr) int {
	return n.Swarm.MTU(ctx, target)
}

// INET256FromP2P converts a p2p.SecureSwarm to an networks.Swarm
func INET256FromP2P(x p2p.SecureSwarm) networks.Swarm {
	return swarmWrapper{s: x}
}

type swarmWrapper struct {
	s p2p.SecureSwarm
}

func (s swarmWrapper) Tell(ctx context.Context, dst inet256.Addr, m p2p.IOVec) error {
	return s.s.Tell(ctx, dst, m)
}

func (s swarmWrapper) Receive(ctx context.Context, th inet256.ReceiveFunc) error {
	return s.s.Receive(ctx, func(m p2p.Message) {
		th(inet256.Message{
			Src:     m.Src.(inet256.Addr),
			Dst:     m.Dst.(inet256.Addr),
			Payload: m.Payload,
		})
	})
}

func (s swarmWrapper) LookupPublicKey(ctx context.Context, target inet256.Addr) (inet256.PublicKey, error) {
	return s.s.LookupPublicKey(ctx, target)
}

func (s swarmWrapper) PublicKey() inet256.PublicKey {
	return s.s.PublicKey()
}

func (s swarmWrapper) MTU(ctx context.Context, target inet256.Addr) int {
	return s.s.MTU(ctx, target)
}

func (s swarmWrapper) LocalAddr() inet256.Addr {
	return s.s.LocalAddrs()[0].(inet256.Addr)
}

func (s swarmWrapper) Close() error {
	return s.s.Close()
}

// SwarmFromNetwork converts a networks.Network to networks.Swarm
func SwarmFromNetwork(nw networks.Network) networks.Swarm {
	panic("")
}
