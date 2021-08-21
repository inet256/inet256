package inet256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/sirupsen/logrus"
)

// Message is the essential information carried Tell and Recv
// provided as a struct for use in queues or other APIs
type Message struct {
	Src     Addr
	Dst     Addr
	Payload []byte
}

// Swarm is similar to a p2p.Swarm, but uses inet256.Addrs instead of p2p.Addrs
type Swarm interface {
	Tell(ctx context.Context, dst Addr, m p2p.IOVec) error
	Receive(ctx context.Context, src, dst *Addr, buf []byte) (int, error)

	Close() error
	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	LocalAddr() Addr
}

const (
	// TransportMTU is the guaranteed MTU presented to networks.
	TransportMTU = (1 << 16) - 1

	// MinMTU is the minimum MTU a network can provide to any address.
	// Applications should be designed to operate correctly if they have to send messages up to this size.
	MinMTU = 1 << 15
	// MaxMTU is the largest message that a network will ever receive from any address.
	// Applications should be prepared to receieve this much data at a time or they may encounter io.ErrShortBuffer
	MaxMTU = 1 << 16
)

// Network is a network for sending messages between peers
type Network interface {
	Tell(ctx context.Context, addr Addr, data []byte) error
	Receive(ctx context.Context, src, dst *Addr, buf []byte) (int, error)
	WaitReceive(ctx context.Context) error
	LocalAddr() Addr
	MTU(ctx context.Context, addr Addr) int

	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

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

// RecvNonBlocking calls receive on the network, but immediately errors
// if no data can be returned
func ReceiveNonBlocking(n Network, src, dst *Addr, buf []byte) (int, error) {
	return n.Receive(NonBlockingContext(), src, dst, buf)
}

func NonBlockingContext() context.Context {
	return nonBlockingCtx{}
}

func IsNonBlockingContext(ctx context.Context) bool {
	return ctx == NonBlockingContext()
}

type nonBlockingCtx struct{}

func (ctx nonBlockingCtx) Err() error {
	return ErrWouldBlock
}

func (ctx nonBlockingCtx) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (ctx nonBlockingCtx) Deadline() (time.Time, bool) {
	return time.Time{}, true
}

func (ctx nonBlockingCtx) Value(k interface{}) interface{} {
	return context.Background().Value(k)
}
