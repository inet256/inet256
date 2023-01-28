package mesh256

import (
	"context"
	"errors"
	"math"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/swarmutil"

	"github.com/inet256/inet256/internal/netutil"
)

type chainSwarm[A p2p.Addr, Pub any] struct {
	swarms []p2p.SecureSwarm[A, Pub]
	hub    swarmutil.TellHub[A]
	sg     netutil.ServiceGroup
}

func newChainSwarm[A p2p.Addr, Pub any](bgCtx context.Context, ss ...p2p.SecureSwarm[A, Pub]) p2p.SecureSwarm[A, Pub] {
	cs := &chainSwarm[A, Pub]{
		swarms: ss,
		hub:    swarmutil.NewTellHub[A](),
	}
	cs.sg.Background = bgCtx
	for i := range ss {
		i := i
		cs.sg.Go(func(ctx context.Context) error {
			for {
				if err := ss[i].Receive(ctx, func(m p2p.Message[A]) {
					cs.hub.Deliver(ctx, m)
				}); err != nil {
					return err
				}
			}
		})
	}
	return cs
}

func (n *chainSwarm[A, Pub]) Tell(ctx context.Context, dst A, v p2p.IOVec) (retErr error) {
	for _, sw := range n.swarms {
		if err := sw.Tell(ctx, dst, v); err != nil {
			retErr = err
		} else {
			return nil
		}
	}
	return retErr
}

func (n *chainSwarm[A, Pub]) Receive(ctx context.Context, fn func(p2p.Message[A])) error {
	return n.hub.Receive(ctx, fn)
}

func (n *chainSwarm[A, Pub]) LookupPublicKey(ctx context.Context, x A) (ret Pub, _ error) {
	for _, sw := range n.swarms {
		pubKey, err := sw.LookupPublicKey(ctx, x)
		if err == nil {
			return pubKey, nil
		}
	}
	return ret, p2p.ErrPublicKeyNotFound
}

func (n *chainSwarm[A, Pub]) LocalAddrs() (ret []A) {
	for _, sw := range n.swarms {
		ret = append(ret, sw.LocalAddrs()...)
	}
	return ret
}

func (n *chainSwarm[A, Pub]) PublicKey() Pub {
	return n.swarms[0].PublicKey()
}

func (n *chainSwarm[A, Pub]) Close() (retErr error) {
	var el netutil.ErrList
	el.Add(n.sg.Stop())
	n.hub.CloseWithError(p2p.ErrClosed)
	for _, sw := range n.swarms {
		el.Add(sw.Close())
	}
	return el.Err()
}

func (n *chainSwarm[A, Pub]) MTU(ctx context.Context, x A) int {
	var min int = math.MaxInt32
	for _, sw := range n.swarms {
		m := sw.MTU(ctx, x)
		if m < min {
			min = m
		}
	}
	return min
}

func (n *chainSwarm[A, Pub]) MaxIncomingSize() int {
	var max int = math.MinInt32
	for _, sw := range n.swarms {
		m := sw.MaxIncomingSize()
		if m > max {
			max = m
		}
	}
	return max
}

func (n *chainSwarm[A, Pub]) ParseAddr(x []byte) (ret A, retErr error) {
	for _, sw := range n.swarms {
		ret, retErr = sw.ParseAddr(x)
		if retErr == nil {
			return ret, nil
		}
	}
	if retErr != nil {
		return ret, retErr
	}
	return ret, errors.New("chainSwarm: could not parse address")
}
