package mesh256

import (
	"context"
	"errors"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/swarmutil"
	"github.com/inet256/inet256/pkg/inet256"
)

type loopbackAddr interface {
	p2p.Addr
	comparable
}

// loopbackSwarm adds loopback behavior to a swarm.
type loopbackSwarm[A loopbackAddr, Pub any] struct {
	localAddr A
	localKey  Pub
	hub       *swarmutil.TellHub[A]
}

func newLoopbackSwarm[A loopbackAddr, Pub any](local A, publicKey Pub) *loopbackSwarm[A, Pub] {
	return &loopbackSwarm[A, Pub]{
		localAddr: local,
		localKey:  publicKey,
		hub:       swarmutil.NewTellHub[A](),
	}
}

func (n *loopbackSwarm[A, Pub]) Tell(ctx context.Context, dst A, v p2p.IOVec) error {
	if dst != n.localAddr {
		return errors.New("loopback: address unreachable")
	}
	return n.hub.Deliver(ctx, p2p.Message[A]{
		Src:     dst,
		Dst:     dst,
		Payload: p2p.VecBytes(nil, v),
	})
}

func (n *loopbackSwarm[A, Pub]) Receive(ctx context.Context, fn func(p2p.Message[A])) error {
	return n.hub.Receive(ctx, fn)
}

func (n *loopbackSwarm[A, Pub]) LocalAddrs() []A {
	return []A{n.localAddr}
}

func (n *loopbackSwarm[A, Pub]) MTU(ctx context.Context, x A) int {
	return n.MaxIncomingSize()
}

func (n *loopbackSwarm[A, Pub]) LookupPublicKey(ctx context.Context, addr A) (ret Pub, _ error) {
	if addr == n.localAddr {
		return n.localKey, nil
	}
	return ret, p2p.ErrPublicKeyNotFound
}

func (n *loopbackSwarm[A, Pub]) PublicKey() Pub {
	return n.localKey
}

func (n *loopbackSwarm[A, Pub]) Close() error {
	return nil
}

func (n *loopbackSwarm[A, Pub]) ParseAddr(x []byte) (A, error) {
	panic("not implemented")
}

func (n *loopbackSwarm[A, Pub]) MaxIncomingSize() int {
	return inet256.MaxMTU
}
