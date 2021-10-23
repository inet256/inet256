package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
)

const (
	// MinMTU is the minimum MTU a network can provide to any address.
	// Applications should be designed to operate correctly if they can only send messages up to this size.
	MinMTU = 1 << 15
	// MaxMTU is the size of largest message that a network will ever receive from any address.
	// Applications should be prepared to receieve this much data at a time or they may encounter io.ErrShortBuffer
	MaxMTU = 1 << 16
)

// Message is the essential information carried by Tell and Receive
// provided as a struct for use in queues or other APIs
type Message struct {
	Src     Addr
	Dst     Addr
	Payload []byte
}

// ReceiveFunc is passed as a callback to Node.Receive
type ReceiveFunc = func(Message)

// Node is a single host in an INET256 network.
// Nodes send and receive messages to and from other nodes in the network.
// Nodes are usually created and managed by a Service.
// Nodes have an single ID or address corresponding to their public key.
//
// This interface is compatible with the INET256 specification.
type Node interface {
	Tell(ctx context.Context, addr Addr, data []byte) error
	Receive(ctx context.Context, fn ReceiveFunc) error

	MTU(ctx context.Context, addr Addr) int
	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

	LocalAddr() Addr
	PublicKey() PublicKey

	Close() error
}

// Service is the top level INET256 object.
// It manages a set of nodes which can be created and deleted.
//
// This interface is compatible with the INET256 specification.
type Service interface {
	CreateNode(ctx context.Context, privKey p2p.PrivateKey) (Node, error)
	DeleteNode(privKey p2p.PrivateKey) error

	LookupPublicKey(ctx context.Context, addr Addr) (p2p.PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)
	MTU(ctx context.Context, addr Addr) int
}

// Receive is a utility method for copying a message from the Node n into msg
func Receive(ctx context.Context, n Node, msg *Message) error {
	return n.Receive(ctx, func(m2 Message) {
		msg.Src = m2.Src
		msg.Dst = m2.Dst
		msg.Payload = append(msg.Payload[:0], m2.Payload...)
	})
}
