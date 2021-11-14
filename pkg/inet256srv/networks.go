package inet256srv

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/inet256/inet256/networks"
	"github.com/inet256/inet256/pkg/inet256"
	"github.com/inet256/inet256/pkg/netutil"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

const (
	TransportMTU = networks.TransportMTU

	MinMTU = inet256.MinMTU
	MaxMTU = inet256.MaxMTU
)

// multiNetwork presents multiple networks as a single network
type multiNetwork struct {
	networks []Network
	addrMap  sync.Map
	sg       *netutil.ServiceGroup
	hub      *netutil.TellHub
}

func newMultiNetwork(networks ...Network) Network {
	for i := 0; i < len(networks)-1; i++ {
		if networks[i].LocalAddr() != networks[i+1].LocalAddr() {
			panic("network addresses do not match")
		}
	}
	hub := netutil.NewTellHub()
	sg := &netutil.ServiceGroup{}
	for i := range networks {
		sg.Go(func(ctx context.Context) error {
			for {
				if err := networks[i].Receive(ctx, func(m inet256.Message) {
					hub.Deliver(ctx, m)
				}); err != nil {
					return err
				}
			}
		})
	}
	return &multiNetwork{
		networks: networks,
		sg:       sg,
		hub:      hub,
	}
}

func (mn *multiNetwork) Tell(ctx context.Context, dst Addr, data []byte) error {
	network, err := mn.whichNetwork(ctx, dst)
	if err != nil {
		return err
	}
	return network.Tell(ctx, dst, data)
}

func (mn *multiNetwork) Receive(ctx context.Context, fn func(inet256.Message)) error {
	return mn.hub.Receive(ctx, fn)
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

func (mn *multiNetwork) PublicKey() inet256.PublicKey {
	return mn.networks[0].PublicKey()
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
	var el netutil.ErrList
	el.Do(mn.sg.Stop)
	mn.hub.CloseWithError(inet256.ErrClosed)
	for _, n := range mn.networks {
		el.Do(n.Close)
	}
	return el.Err()
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
	return Addr{}, nil, fmt.Errorf("errors occurred %v", errs)
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
	hub      *netutil.TellHub
	sg       *netutil.ServiceGroup
}

func newChainNetwork(ns ...Network) Network {
	hub := netutil.NewTellHub()
	sg := &netutil.ServiceGroup{}
	for i := range ns {
		i := i
		sg.Go(func(ctx context.Context) error {
			for {
				if err := ns[i].Receive(ctx, func(m inet256.Message) {
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

func (n chainNetwork) Receive(ctx context.Context, fn func(inet256.Message)) error {
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

func (n chainNetwork) PublicKey() inet256.PublicKey {
	return n.networks[0].PublicKey()
}

func (n chainNetwork) Close() (retErr error) {
	var el netutil.ErrList
	el.Do(n.sg.Stop)
	n.hub.CloseWithError(inet256.ErrClosed)
	for _, n2 := range n.networks {
		el.Do(n2.Close)
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
}

func newLoopbackNetwork(localKey p2p.PublicKey) Network {
	ln := &loopbackNetwork{
		localAddr: inet256.NewAddr(localKey),
		localKey:  localKey,
		hub:       netutil.NewTellHub(),
	}
	return ln
}

func (n *loopbackNetwork) Tell(ctx context.Context, dst Addr, data []byte) error {
	if dst != n.localAddr {
		return inet256.ErrAddrUnreachable{Addr: dst}
	}
	return n.hub.Deliver(ctx, netutil.Message{Src: dst, Dst: dst, Payload: data})
}

func (n *loopbackNetwork) Receive(ctx context.Context, fn func(inet256.Message)) error {
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
