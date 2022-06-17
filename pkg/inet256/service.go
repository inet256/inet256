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
	// Tell sends a message containing data to the node at addr.
	// The message will be delivered at most once.
	Send(ctx context.Context, addr Addr, data []byte) error
	// Receive calls fn with a message sent to this node.
	// The message fields, and payload must not be accessed outside fn.
	Receive(ctx context.Context, fn ReceiveFunc) error

	// MTU finds the maximum message size that can be sent to addr.
	// If the context expires, a reasonable default (normally a significant underestimate) will be returned.
	MTU(ctx context.Context, addr Addr) int
	// LookupPublicKey attempts to find the public key corresponding to addr.
	// If it can't find it, ErrPublicKeyNotFound is returned.
	LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error)
	// FindAddr looks for an address with nbits leading bits in common with prefix.
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

	// LocalAddr returns this Node's address
	LocalAddr() Addr
	// PublicKey returns this Node's public key
	PublicKey() PublicKey

	// Close indicates no more messages should be sent or received from this node
	// and releases any resources allocated for this node.
	Close() error
}

// Service is the top level INET256 object.
// It manages a set of nodes which can be created and deleted.
//
// This interface is compatible with the INET256 specification.
type Service interface {
	Open(ctx context.Context, privKey p2p.PrivateKey, opts ...NodeOption) (Node, error)
	Drop(ctx context.Context, privKey p2p.PrivateKey) error
}

// NodeOption is the type of functions which configure a Node.
type NodeOption = func(*NodeConfig)

// NodeConfig is an aggregate of applied NodeOptions
// Not all implementations will support all the options.
type NodeConfig struct {
}

// CollectNodeOptions applyies each option in opts to a NodeOptions
// and returns the result.
func CollectNodeOptions(opts []NodeOption) (cfg NodeConfig) {
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Receive is a utility function for copying a message from the Node n into msg
func Receive(ctx context.Context, n Node, msg *Message) error {
	return n.Receive(ctx, func(m2 Message) {
		msg.Src = m2.Src
		msg.Dst = m2.Dst
		msg.Payload = append(msg.Payload[:0], m2.Payload...)
	})
}
