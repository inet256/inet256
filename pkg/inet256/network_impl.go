package inet256

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// multiNetwork presents multiple networks as a single network
type multiNetwork struct {
	networks []Network
	addrMap  sync.Map
	selector *Selector
}

func newMultiNetwork(networks ...Network) Network {
	for i := 0; i < len(networks)-1; i++ {
		if networks[i].LocalAddr() != networks[i+1].LocalAddr() {
			panic("network addresses do not match")
		}
	}
	mn := &multiNetwork{
		networks: networks,
		selector: NewSelector(networks),
	}
	return mn
}

func (mn *multiNetwork) Tell(ctx context.Context, dst Addr, data []byte) error {
	network, err := mn.whichNetwork(ctx, dst)
	if err != nil {
		return err
	}
	return network.Tell(ctx, dst, data)
}

func (mn *multiNetwork) Recv(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	return mn.selector.Recv(ctx, src, dst, buf)
}

func (mn *multiNetwork) WaitRecv(ctx context.Context) error {
	return mn.selector.Wait(ctx)
}

func (mn *multiNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	addr, _, err := mn.addrWithPrefix(ctx, prefix, nbits)
	return addr, err
}

func (mn *multiNetwork) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	network, err := mn.whichNetwork(ctx, target)
	if err != nil {
		return nil, err
	}
	return network.LookupPublicKey(ctx, target)
}

func (mn *multiNetwork) MTU(ctx context.Context, target Addr) int {
	network, err := mn.whichNetwork(ctx, target)
	if err != nil {
		return MinMTU
	}
	return network.MTU(ctx, target)
}

func (mn *multiNetwork) LocalAddr() Addr {
	return mn.networks[0].LocalAddr()
}

func (mn *multiNetwork) Bootstrap(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, n := range mn.networks {
		n := n
		eg.Go(func() error {
			return n.Bootstrap(ctx)
		})
	}
	return eg.Wait()
}

func (mn *multiNetwork) Close() (retErr error) {
	mn.selector.Close()
	for _, n := range mn.networks {
		if err := n.Close(); err != nil {
			logrus.Error(err)
			if retErr == nil {
				retErr = err
			}
		}
	}
	return retErr
}

func (mn *multiNetwork) addrWithPrefix(ctx context.Context, prefix []byte, nbits int) (addr Addr, network Network, err error) {
	var cf context.CancelFunc
	if deadline, ok := ctx.Deadline(); ok {
		now := time.Now()
		ctx, cf = context.WithTimeout(ctx, deadline.Sub(now)/3)
	} else {
		ctx, cf = context.WithCancel(ctx)
	}
	defer cf()

	addrs := make([]Addr, len(mn.networks))
	errs := make([]error, len(mn.networks))
	wg := sync.WaitGroup{}
	wg.Add(len(mn.networks))
	for i, network := range mn.networks[:] {
		i := i
		network := network
		go func() {
			addrs[i], errs[i] = network.FindAddr(ctx, prefix, nbits)
			if errs[i] == nil {
				cf()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	for i, addr := range addrs {
		if errs[i] == nil {
			return addr, mn.networks[i], nil
		}
	}
	return Addr{}, nil, fmt.Errorf("errors occurred %v", addrs)
}

func (mn *multiNetwork) whichNetwork(ctx context.Context, addr Addr) (Network, error) {
	x, exists := mn.addrMap.Load(addr)
	if exists {
		return x.(Network), nil
	}
	_, network, err := mn.addrWithPrefix(ctx, addr[:], len(addr)*8)
	if err != nil {
		return nil, errors.Wrap(err, "selecting network")
	}
	mn.addrMap.Store(addr, network)
	return network, nil
}

type chainNetwork struct {
	networks []Network
	selector *Selector
}

func newChainNetwork(ns ...Network) Network {
	return chainNetwork{
		networks: ns,
		selector: NewSelector(ns),
	}
}

func (n chainNetwork) Tell(ctx context.Context, dst Addr, data []byte) (retErr error) {
	for _, n2 := range n.networks {
		if err := n2.Tell(ctx, dst, data); err != nil {
			retErr = err
		} else {
			return nil
		}
	}
	return retErr
}

func (n chainNetwork) Recv(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	return n.selector.Recv(ctx, src, dst, buf)
}

func (n chainNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	for _, n2 := range n.networks {
		addr, err := n2.FindAddr(ctx, prefix, nbits)
		if err == nil {
			return addr, nil
		}
	}
	return Addr{}, ErrNoAddrWithPrefix
}

func (n chainNetwork) LookupPublicKey(ctx context.Context, x Addr) (p2p.PublicKey, error) {
	for _, n2 := range n.networks {
		pubKey, err := n2.LookupPublicKey(ctx, x)
		if err == nil {
			return pubKey, nil
		}
	}
	return nil, ErrPublicKeyNotFound
}

func (n chainNetwork) LocalAddr() Addr {
	return n.networks[0].LocalAddr()
}

func (n chainNetwork) Close() (retErr error) {
	for _, n2 := range n.networks {
		if err := n2.Close(); retErr == nil {
			retErr = err
		}
	}
	if err := n.selector.Close(); retErr == nil {
		retErr = err
	}
	return retErr
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

func (n chainNetwork) WaitRecv(ctx context.Context) error {
	return n.selector.Wait(ctx)
}

type loopbackNetwork struct {
	localAddr Addr
	localKey  p2p.PublicKey
	hub       *TellHub
}

func newLoopbackNetwork(localKey p2p.PublicKey) Network {
	ln := &loopbackNetwork{
		localAddr: p2p.NewPeerID(localKey),
		localKey:  localKey,
		hub:       NewTellHub(),
	}
	return ln
}

func (n *loopbackNetwork) Tell(ctx context.Context, dst Addr, data []byte) error {
	if dst != n.localAddr {
		return ErrAddrUnreachable{Addr: dst}
	}
	return n.hub.Deliver(ctx, Message{Src: dst, Dst: dst, Payload: data})
}

func (n *loopbackNetwork) Recv(ctx context.Context, src, dst *Addr, buf []byte) (int, error) {
	return n.hub.Recv(ctx, src, dst, buf)
}

func (n *loopbackNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	if HasPrefix(n.localAddr[:], prefix, nbits) {
		return n.localAddr, nil
	}
	return Addr{}, ErrNoAddrWithPrefix
}

func (n *loopbackNetwork) LocalAddr() Addr {
	return n.localAddr
}

func (n *loopbackNetwork) Close() error {
	return nil
}

func (n *loopbackNetwork) WaitRecv(ctx context.Context) error {
	return n.hub.Wait(ctx)
}

func (n *loopbackNetwork) MTU(context.Context, Addr) int {
	return 1 << 20
}

func (n *loopbackNetwork) Bootstrap(ctx context.Context) error {
	return nil
}

func (n *loopbackNetwork) LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error) {
	if addr == n.localAddr {
		return n.localKey, nil
	}
	return nil, ErrPublicKeyNotFound
}
