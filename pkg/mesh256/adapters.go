package mesh256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
)

// swarmFromP2P converts a p2p.SecureSwarm to an networks.Swarm
func swarmFromP2P(x p2p.SecureSwarm[inet256.Addr]) Swarm {
	return swarmWrapper{x}
}

type swarmWrapper struct {
	p2p.SecureSwarm[inet256.Addr]
}

func (s swarmWrapper) LocalAddr() inet256.Addr {
	return s.SecureSwarm.LocalAddrs()[0]
}

type extraSwarmMethods struct{}

func (extraSwarmMethods) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (extraSwarmMethods) ParseAddr(data []byte) (inet256.Addr, error) {
	var addr inet256.Addr
	if err := addr.UnmarshalText(data); err != nil {
		return inet256.Addr{}, err
	}
	return addr, nil
}

type FindAddrFunc = func(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)

type BootstrapFunc = func(ctx context.Context) error

var _ Network = &swarmAdapter{}

func networkFromP2PSwarm(x p2p.SecureSwarm[inet256.Addr], findAddr FindAddrFunc, bootstrapFunc BootstrapFunc) Network {
	return NetworkFromSwarm(swarmWrapper{x}, findAddr, bootstrapFunc)
}

// NetworkFromSwarm creates a Network from a Swarm
func NetworkFromSwarm(x Swarm, findAddr FindAddrFunc, boorstrapFunc BootstrapFunc) Network {
	return &swarmAdapter{
		Swarm:         x,
		findAddr:      findAddr,
		bootstrapFunc: boorstrapFunc,
	}
}

type swarmAdapter struct {
	Swarm
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

type p2pSwarm struct {
	Swarm
	extraSwarmMethods
}

func p2pSwarmFromNetwork(x Network) p2p.SecureSwarm[inet256.Addr] {
	return p2pSwarm{Swarm: x}
}

func (s p2pSwarm) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{s.Swarm.LocalAddr()}
}
