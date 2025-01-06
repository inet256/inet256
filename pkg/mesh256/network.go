package mesh256

import (
	"context"

	"go.brendoncarroll.net/p2p"
	"go.brendoncarroll.net/p2p/f/x509"
	"go.inet256.org/inet256/pkg/inet256"
)

// Network is an instantiated network routing algorithm
//
// This interface is not described in the spec, and is incidental to the implementation.
type Network interface {
	p2p.Teller[inet256.Addr]
	p2p.Receiver[inet256.Addr]
	p2p.Secure[inet256.Addr, inet256.PublicKey]

	LocalAddr() inet256.Addr
	MTU() int
	Close() error

	FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)
}

// Swarm is similar to a p2p.Swarm, but uses inet256.Addrs instead of p2p.Addrs
//
// This interface is not described in the spec, and is incidental to the implementation.
type Swarm interface {
	p2p.Teller[inet256.Addr]
	p2p.Receiver[inet256.Addr]
	p2p.Secure[inet256.Addr, inet256.PublicKey]

	LocalAddr() Addr
	MTU() int
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

var _ Swarm = swarm{}

// swarm is presented to Networks
type swarm struct {
	p2p.SecureSwarm[inet256.Addr, x509.PublicKey]
}

func (s swarm) PublicKey() inet256.PublicKey {
	pubKey, err := PublicKeyFromX509(s.SecureSwarm.PublicKey())
	if err != nil {
		panic(err)
	}
	return pubKey
}

func (s swarm) LookupPublicKey(ctx context.Context, target inet256.Addr) (ret inet256.PublicKey, _ error) {
	pub, err := s.SecureSwarm.LookupPublicKey(ctx, target)
	if err != nil {
		return ret, err
	}
	return PublicKeyFromX509(pub)
}

func (s swarm) LocalAddr() inet256.Addr {
	return s.SecureSwarm.LocalAddrs()[0]
}

// wrapNetwork creates a p2p.SecureSwarm[inet256.Addr, x509.PublicKey]
// it secures the underlying network.
// and then wraps it with loopback behavior
func wrapNetwork(privateKey inet256.PrivateKey, x Network) p2p.SecureSwarm[inet256.Addr, x509.PublicKey] {
	insecure := networkSwarm{x}
	idenSw := newIdentitySwarm(privateKey, insecure)

	pubX509 := PublicKeyFromINET256(privateKey.Public())
	localAddr := inet256.NewAddr(privateKey.Public())
	lbSw := newLoopbackSwarm(localAddr, pubX509)

	return newChainSwarm[inet256.Addr, x509.PublicKey](context.TODO(), lbSw, idenSw)
}

// networkSwarm converts a Network to a p2p.Swarm[inet256.Addr, inet256.PublicKey]
type networkSwarm struct {
	Network
}

func (s networkSwarm) LocalAddrs() []inet256.Addr {
	return []inet256.Addr{s.Network.LocalAddr()}
}

func (s networkSwarm) LookupPublicKey() {
	// not a SecureSwarm
}

func (s networkSwarm) PublicKey() {
	// not a SecureSwarm
}

func (networkSwarm) ParseAddr(data []byte) (inet256.Addr, error) {
	var addr inet256.Addr
	if err := addr.UnmarshalText(data); err != nil {
		return inet256.Addr{}, err
	}
	return addr, nil
}
