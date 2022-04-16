package p2padapter

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
)

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
func INET256FromP2P(x p2p.SecureSwarm[inet256.Addr]) networks.Swarm {
	return swarmWrapper{x}
}

type swarmWrapper struct {
	p2p.SecureSwarm[inet256.Addr]
}

func (s swarmWrapper) LocalAddr() inet256.Addr {
	return s.SecureSwarm.LocalAddrs()[0]
}

type p2pSwarm struct {
	networks.Swarm
	ExtraSwarmMethods
}

// P2PFromINET256 converts a networks.Swarm into a p2p.SecureSwarm
func P2PFromINET256(x networks.Swarm) p2p.SecureSwarm[inet256.Addr] {
	return p2pSwarm{Swarm: x}
}

func (s p2pSwarm) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{s.Swarm.LocalAddr()}
}

// SwarmFromNetwork converts a networks.Network to networks.Swarm
func P2PSwarmFromNode(node inet256.Node) p2p.SecureSwarm[inet256.Addr] {
	return p2pNode{Node: node}
}

type p2pNode struct {
	inet256.Node
	ExtraSwarmMethods
}

func (pn p2pNode) Tell(ctx context.Context, dst inet256.Addr, v p2p.IOVec) error {
	return pn.Node.Send(ctx, dst, p2p.VecBytes(nil, v))
}

func (pn p2pNode) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return pn.Node.Receive(ctx, func(msg inet256.Message) {
		fn(p2p.Message[inet256.Addr]{
			Src:     msg.Src,
			Dst:     msg.Dst,
			Payload: msg.Payload,
		})
	})
}

func (pn p2pNode) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{pn.Node.LocalAddr()}
}

type ExtraSwarmMethods struct{}

func (ExtraSwarmMethods) MaxIncomingSize() int {
	return inet256.MaxMTU
}

func (ExtraSwarmMethods) ParseAddr(data []byte) (inet256.Addr, error) {
	var addr inet256.Addr
	if err := addr.UnmarshalText(data); err != nil {
		return inet256.Addr{}, err
	}
	return addr, nil
}
