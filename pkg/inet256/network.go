package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/sirupsen/logrus"
)

// Swarm is similar to a p2p.Swarm, but uses inet256.Addrs instead of p2p.Addrs
//
// This interface is not described in the spec, and is incidental to the implementation.
type Swarm interface {
	Tell(ctx context.Context, dst Addr, m p2p.IOVec) error
	Receive(ctx context.Context, th ReceiveFunc) error

	Close() error
	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	PublicKey() PublicKey
	LocalAddr() Addr
}

const (
	// TransportMTU is the guaranteed MTU presented to networks.
	TransportMTU = (1 << 16) - 1
)

// Network is an instantiated network routing algorithm
//
// This interface is not described in the spec, and is incidental to the implementation.
type Network interface {
	Tell(ctx context.Context, addr Addr, data []byte) error
	Receive(ctx context.Context, fn ReceiveFunc) error

	MTU(ctx context.Context, addr Addr) int
	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

	LocalAddr() Addr
	PublicKey() PublicKey

	Bootstrap(ctx context.Context) error
	Close() error
}

// NetworkParams are passed to a NetworkFactory to create a Network.
// This type really defines the problem domain quite well. Essentially
// it is a set of one-hop peers and a means to send messages to them.
type NetworkParams struct {
	PrivateKey PrivateKey
	Swarm      Swarm
	Peers      PeerSet

	Logger *Logger
}

// NetworkFactory is a constructor for a network
type NetworkFactory func(NetworkParams) Network

type Logger = logrus.Logger
