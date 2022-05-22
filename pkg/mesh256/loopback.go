package mesh256

import (
	"context"
	"math"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"golang.org/x/sync/errgroup"
)

type chainNetwork struct {
	networks []Network
	hub      *netutil.TellHub
	sg       *netutil.ServiceGroup
	extraSwarmMethods
}

func newChainNetwork(ns ...Network) Network {
	hub := netutil.NewTellHub()
	sg := &netutil.ServiceGroup{}
	for i := range ns {
		i := i
		sg.Go(func(ctx context.Context) error {
			for {
				if err := ns[i].Receive(ctx, func(m p2p.Message[inet256.Addr]) {
					hub.Deliver(ctx, m)
				}); err != nil {
					return err
				}
			}
		})
	}
	return chainNetwork{
		networks: ns,
		hub:      hub,
		sg:       sg,
	}
}

func (n chainNetwork) Tell(ctx context.Context, dst Addr, v p2p.IOVec) (retErr error) {
	for _, n2 := range n.networks {
		if err := n2.Tell(ctx, dst, v); err != nil {
			retErr = err
		} else {
			return nil
		}
	}
	return retErr
}

func (n chainNetwork) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return n.hub.Receive(ctx, fn)
}

func (n chainNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	for _, n2 := range n.networks {
		addr, err := n2.FindAddr(ctx, prefix, nbits)
		if err == nil {
			return addr, nil
		}
	}
	return Addr{}, inet256.ErrNoAddrWithPrefix
}

func (n chainNetwork) LookupPublicKey(ctx context.Context, x Addr) (p2p.PublicKey, error) {
	for _, n2 := range n.networks {
		pubKey, err := n2.LookupPublicKey(ctx, x)
		if err == nil {
			return pubKey, nil
		}
	}
	return nil, inet256.ErrPublicKeyNotFound
}

func (n chainNetwork) LocalAddr() Addr {
	return n.networks[0].LocalAddr()
}

func (n chainNetwork) LocalAddrs() []Addr {
	return []Addr{n.LocalAddr()}
}

func (n chainNetwork) PublicKey() inet256.PublicKey {
	return n.networks[0].PublicKey()
}

func (n chainNetwork) Close() (retErr error) {
	var el netutil.ErrList
	el.Add(n.sg.Stop())
	n.hub.CloseWithError(inet256.ErrClosed)
	for _, n2 := range n.networks {
		el.Add(n2.Close())
	}
	return el.Err()
}

func (n chainNetwork) MTU(ctx context.Context, x Addr) int {
	var min int = math.MaxInt32
	for _, n2 := range n.networks {
		m := n2.MTU(ctx, x)
		if m < min {
			min = m
		}
	}
	return min
}

func (n chainNetwork) Bootstrap(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, n2 := range n.networks {
		n2 := n2
		eg.Go(func() error {
			return n2.Bootstrap(ctx)
		})
	}
	return eg.Wait()
}

type loopbackNetwork struct {
	localAddr Addr
	localKey  p2p.PublicKey
	hub       *netutil.TellHub
	extraSwarmMethods
}

func newLoopbackNetwork(localKey p2p.PublicKey) Network {
	ln := &loopbackNetwork{
		localAddr: inet256.NewAddr(localKey),
		localKey:  localKey,
		hub:       netutil.NewTellHub(),
	}
	return ln
}

func (n *loopbackNetwork) Tell(ctx context.Context, dst Addr, v p2p.IOVec) error {
	if dst != n.localAddr {
		return inet256.ErrAddrUnreachable{Addr: dst}
	}
	return n.hub.Deliver(ctx, p2p.Message[inet256.Addr]{
		Src:     dst,
		Dst:     dst,
		Payload: p2p.VecBytes(nil, v),
	})
}

func (n *loopbackNetwork) Receive(ctx context.Context, fn func(p2p.Message[inet256.Addr])) error {
	return n.hub.Receive(ctx, fn)
}

func (n *loopbackNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	if inet256.HasPrefix(n.localAddr[:], prefix, nbits) {
		return n.localAddr, nil
	}
	return Addr{}, inet256.ErrNoAddrWithPrefix
}

func (n *loopbackNetwork) LocalAddr() Addr {
	return n.localAddr
}

func (n *loopbackNetwork) LocalAddrs() []Addr {
	return []Addr{n.LocalAddr()}
}

func (n *loopbackNetwork) PublicKey() inet256.PublicKey {
	return n.localKey
}

func (n *loopbackNetwork) Close() error {
	n.hub.CloseWithError(nil)
	return nil
}

func (n *loopbackNetwork) MTU(context.Context, Addr) int {
	return 1 << 20
}

func (n *loopbackNetwork) Bootstrap(ctx context.Context) error {
	return nil
}

func (n *loopbackNetwork) LookupPublicKey(ctx context.Context, addr Addr) (inet256.PublicKey, error) {
	if addr == n.localAddr {
		return n.localKey, nil
	}
	return nil, inet256.ErrPublicKeyNotFound
}
