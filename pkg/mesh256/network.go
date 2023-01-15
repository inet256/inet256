package mesh256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/f/x509"
	"github.com/inet256/inet256/pkg/inet256"
)

// Network is an instantiated network routing algorithm
//
// This interface is not described in the spec, and is incidental to the implementation.
type Network interface {
	p2p.Teller[inet256.Addr]
	LocalAddr() inet256.Addr
	MTU(context.Context, inet256.Addr) int
	Close() error

	LookupPublicKey(context.Context, inet256.Addr) (inet256.PublicKey, error)
	PublicKey() inet256.PublicKey

	FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)
}

// Swarm is similar to a p2p.Swarm, but uses inet256.Addrs instead of p2p.Addrs
//
// This interface is not described in the spec, and is incidental to the implementation.
type Swarm interface {
	p2p.Teller[inet256.Addr]

	LookupPublicKey(context.Context, inet256.Addr) (inet256.PublicKey, error)
	PublicKey() inet256.PublicKey

	LocalAddr() Addr
	MTU(ctx context.Context, addr Addr) int
	Close() error
}

// Params are passed to a NetworkFactory to create a Network.
// This type really defines the problem domain quite well. Essentially
// it is a set of one-hop peers and a means to send messages to them.
type NetworkParams struct {
	// PrivateKey is the private signing key used for proving identity on the network.
	PrivateKey inet256.PrivateKey
	// Swarm facillitates communication with adjacent Nodes.
	Swarm Swarm
	// Peers is the set of peers to communicate with.
	Peers PeerSet
	// Background is the context that should be used for background operations.
	// It enables instrumentation through the stdctx/logctx package.
	Background context.Context
}

// NetworkFactory is a constructor for a network
type NetworkFactory func(NetworkParams) Network

// wrapNetwork creates a p2p.SecureSwarm[inet256.Addr, x509.PublicKey]
// it secures the underlying network.
// and then wraps it with loopback behavior
func wrapNetwork(privateKey inet256.PrivateKey, x Network) p2p.SecureSwarm[inet256.Addr, x509.PublicKey] {
	insecure := p2pSwarmFromNetwork(x)
	idenSw := newIdentitySwarm(privateKey, insecure)

	pubX509 := PublicKeyFromINET256(privateKey.Public())
	localAddr := inet256.NewAddr(privateKey.Public())
	lbSw := newLoopbackSwarm(localAddr, pubX509)

	return newChainSwarm[inet256.Addr, x509.PublicKey](lbSw, idenSw)
}

type FindAddrFunc = func(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)

var _ Network = &networkFromSwarm{}

// networkFromSwarm implements a Network from a Swarm
type networkFromSwarm struct {
	Swarm
	findAddr FindAddrFunc
}

func newNetworkFromP2PSwarm(x p2p.SecureSwarm[inet256.Addr, inet256.PublicKey], findAddr FindAddrFunc) Network {
	return &networkFromSwarm{
		Swarm:    swarmWrapper{x},
		findAddr: findAddr,
	}
}

func (n *networkFromSwarm) FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error) {
	return n.findAddr(ctx, prefix, nbits)
}

func (n *networkFromSwarm) MTU(ctx context.Context, target inet256.Addr) int {
	return n.Swarm.MTU(ctx, target)
}

// NetworkFromSwarm creates a Network from a Swarm
func NetworkFromSwarm(x Swarm, findAddr FindAddrFunc) Network {
	return &networkFromSwarm{
		Swarm:    x,
		findAddr: findAddr,
	}
}

// p2pSwarm converts a Swarm to a p2p.Swarm
type p2pSwarm struct {
	Swarm
	extraSwarmMethods
}

func p2pSwarmFromNetwork(x Network) p2p.SecureSwarm[inet256.Addr, inet256.PublicKey] {
	return p2pSwarm{Swarm: x}
}

func (s p2pSwarm) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{s.Swarm.LocalAddr()}
}

// func (s p2pSwarm) LookupPublicKey(ctx context.Context, target inet256.Addr) (ret inet256.PublicKey, _ error) {
// 	if err != nil {
// 		return ret, err
// 	}
// 	return convertINET256PublicKey(pubKey), nil
// }

// func (s p2pSwarm) PublicKey() x509.PublicKey {
// 	return convertINET256PublicKey(s.Swarm.PublicKey())
// }

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
