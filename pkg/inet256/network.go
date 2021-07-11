package inet256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	MinMTU = 1 << 15
	MaxMTU = 1 << 16
)

var (
	ErrWouldBlock = errors.New("recv would block")
)

func IsErrWouldBlock(err error) bool {
	return err == ErrWouldBlock
}

// Network is a network for sending messages between peers
type Network interface {
	Tell(ctx context.Context, addr Addr, data []byte) error
	Recv(ctx context.Context, src, dst *Addr, buf []byte) (int, error)
	WaitRecv(ctx context.Context) error
	LocalAddr() Addr
	MTU(ctx context.Context, addr Addr) int

	LookupPublicKey(ctx context.Context, addr Addr) (p2p.PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

	Bootstrap(ctx context.Context) error
	Close() error
}

// NetworkParams are passed to a NetworkFactory to create a Network.
// This type really defines the problem domain quite well. Essentially
// it is a set of one-hop peers and a means to send messages to them.
type NetworkParams struct {
	PrivateKey p2p.PrivateKey
	Swarm      PeerSwarm
	Peers      PeerSet

	Logger *Logger
}

// NetworkFactory is a constructor for a network
type NetworkFactory func(NetworkParams) Network

// NetworkSpec is a name associated with a network factory
type NetworkSpec struct {
	Index   uint64
	Name    string
	Factory NetworkFactory
}

type Logger = logrus.Logger

func RecvNonBlocking(n Network, src, dst *Addr, buf []byte) (int, error) {
	return n.Recv(NonBlockingContext(), src, dst, buf)
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
