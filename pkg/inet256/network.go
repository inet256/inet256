package inet256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/sirupsen/logrus"
)

// Message is the essential information carried by Tell and Receive
// provided as a struct for use in queues or other APIs
type Message struct {
	Src     Addr
	Dst     Addr
	Payload []byte
}

// ReceiveFunc is passed as a callback to Swarm.Receive
type ReceiveFunc = func(Message)

// Swarm is similar to a p2p.Swarm, but uses inet256.Addrs instead of p2p.Addrs
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
	// MinMTU is the minimum MTU a network can provide to any address.
	// Applications should be designed to operate correctly if they can only send messages up to this size.
	MinMTU = 1 << 15
	// MaxMTU is the size of largest message that a network will ever receive from any address.
	// Applications should be prepared to receieve this much data at a time or they may encounter io.ErrShortBuffer
	MaxMTU = 1 << 16
)

// Network is a network of connected nodes which can send messages to each other
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

// Receive is a utility method for copying a message from the Network n into msg
func Receive(ctx context.Context, n Network, msg *Message) error {
	return n.Receive(ctx, func(m2 Message) {
		msg.Src = m2.Src
		msg.Dst = m2.Dst
		msg.Payload = append(msg.Payload[:0], m2.Payload...)
	})
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
