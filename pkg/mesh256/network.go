package mesh256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/sirupsen/logrus"
)

// Swarm is similar to a p2p.Swarm, but uses inet256.Addrs instead of p2p.Addrs
//
// This interface is not described in the spec, and is incidental to the implementation.
type Swarm interface {
	p2p.Teller[inet256.Addr]
	Close() error
	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	PublicKey() PublicKey
	LocalAddr() Addr
	MTU(ctx context.Context, addr Addr) int
}

// Network is an instantiated network routing algorithm
//
// This interface is not described in the spec, and is incidental to the implementation.
type Network interface {
	p2p.Teller[inet256.Addr]
	p2p.Secure[inet256.Addr]
	LocalAddr() inet256.Addr
	MTU(context.Context, inet256.Addr) int
	FindAddr(ctx context.Context, prefix []byte, nbits int) (inet256.Addr, error)

	Bootstrap(ctx context.Context) error
	Close() error
}

// Params are passed to a NetworkFactory to create a Network.
// This type really defines the problem domain quite well. Essentially
// it is a set of one-hop peers and a means to send messages to them.
type NetworkParams struct {
	PrivateKey PrivateKey
	Swarm      Swarm
	Peers      PeerSet

	Logger Logger
}

// NetworkFactory is a constructor for a network
type NetworkFactory func(NetworkParams) Network

type Logger = logrus.FieldLogger
