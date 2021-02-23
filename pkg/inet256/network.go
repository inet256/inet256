package inet256

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// RecvFunc can be passed to Network.OnRecv as a callback to receive messages
type RecvFunc func(src, dst Addr, data []byte)

// NoOpRecvFunc does nothing
func NoOpRecvFunc(src, dst Addr, data []byte) {}

// Network is a network for sending messages between peers
type Network interface {
	Tell(ctx context.Context, addr Addr, data []byte) error
	OnRecv(fn RecvFunc)
	LocalAddr() Addr
	MTU(ctx context.Context, addr Addr) int

	LookupPublicKey(ctx context.Context, addr Addr) (p2p.PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

	WaitReady(ctx context.Context) error
	Close() error
}

// NetworkParams are passed to a NetworkFactory to create a Network.
// This type really defines the problem domain quite well. Essentially
// it is a set of one-hop peers and a means to send messages to them.
type NetworkParams struct {
	PrivateKey p2p.PrivateKey
	Swarm      PeerSwarm
	Peers      PeerSet

	Logger *logrus.Logger
}

// NetworkFactory is a constructor for a network
type NetworkFactory func(NetworkParams) Network

// NetworkSpec is a name associated with a network factory
type NetworkSpec struct {
	Index   uint64
	Name    string
	Factory NetworkFactory
}

type multiNetwork struct {
	networks []Network

	mu      sync.Mutex
	addrMap sync.Map
}

func newMultiNetwork(networks ...Network) Network {
	for i := 0; i < len(networks)-1; i++ {
		if networks[i].LocalAddr() != networks[i+1].LocalAddr() {
			panic("network addresses do not match")
		}
	}
	return &multiNetwork{
		networks: networks,
	}
}

func (mn *multiNetwork) Tell(ctx context.Context, dst Addr, data []byte) error {
	network := mn.whichNetwork(ctx, dst)
	if network == nil {
		return ErrAddrUnreachable{dst}
	}
	return network.Tell(ctx, dst, data)
}

func (mn *multiNetwork) OnRecv(fn RecvFunc) {
	for _, n := range mn.networks {
		n.OnRecv(fn)
	}
}

func (mn *multiNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	addr, _, err := mn.addrWithPrefix(ctx, prefix, nbits)
	return addr, err
}

func (mn *multiNetwork) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	network := mn.whichNetwork(ctx, target)
	if network == nil {
		return nil, ErrAddrUnreachable{target}
	}
	return network.LookupPublicKey(ctx, target)
}

func (mn *multiNetwork) MTU(ctx context.Context, target Addr) int {
	ntwk := mn.whichNetwork(ctx, target)
	return ntwk.MTU(ctx, target)
}

func (mn *multiNetwork) LocalAddr() Addr {
	return mn.networks[0].LocalAddr()
}

func (mn *multiNetwork) WaitReady(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, n := range mn.networks {
		n := n
		eg.Go(func() error {
			return n.WaitReady(ctx)
		})
	}
	return eg.Wait()
}

func (mn *multiNetwork) Close() (retErr error) {
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
	addrs := make([]Addr, len(mn.networks))
	errs := make([]error, len(mn.networks))
	ctx, cf := context.WithCancel(ctx)
	defer cf()
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

func (mn *multiNetwork) whichNetwork(ctx context.Context, addr Addr) Network {
	x, exists := mn.addrMap.Load(addr)
	if exists {
		return x.(Network)
	}
	_, network, err := mn.addrWithPrefix(ctx, addr[:], len(addr)*8)
	if err != nil {
		return nil
	}
	mn.addrMap.Store(addr, network)
	return network
}

type chainNetwork []Network

func newChainNetwork(ns ...Network) Network {
	return chainNetwork(ns)
}

func (n chainNetwork) Tell(ctx context.Context, dst Addr, data []byte) (retErr error) {
	for _, n2 := range n {
		if err := n2.Tell(ctx, dst, data); err != nil {
			retErr = err
		} else {
			return nil
		}
	}
	return retErr
}

func (n chainNetwork) OnRecv(fn RecvFunc) {
	for _, n2 := range n {
		n2.OnRecv(fn)
	}
}

func (n chainNetwork) FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error) {
	for _, n2 := range n {
		addr, err := n2.FindAddr(ctx, prefix, nbits)
		if err == nil {
			return addr, nil
		}
	}
	return Addr{}, ErrNoAddrWithPrefix
}

func (n chainNetwork) LookupPublicKey(ctx context.Context, x Addr) (p2p.PublicKey, error) {
	for _, n2 := range n {
		pubKey, err := n2.LookupPublicKey(ctx, x)
		if err == nil {
			return pubKey, nil
		}
	}
	return nil, ErrPublicKeyNotFound
}

func (n chainNetwork) LocalAddr() Addr {
	return n[0].LocalAddr()
}

func (n chainNetwork) Close() (retErr error) {
	for _, n2 := range n {
		if err := n2.Close(); retErr == nil {
			retErr = err
		}
	}
	return retErr
}

func (n chainNetwork) MTU(ctx context.Context, x Addr) int {
	var min int = math.MaxInt32
	for _, n2 := range n {
		m := n2.MTU(ctx, x)
		if m < min {
			min = m
		}
	}
	return min
}

func (n chainNetwork) WaitReady(ctx context.Context) error {
	eg := errgroup.Group{}
	for _, n2 := range n {
		n2 := n2
		eg.Go(func() error {
			return n2.WaitReady(ctx)
		})
	}
	return eg.Wait()
}

type loopbackNetwork struct {
	localAddr Addr
	localKey  p2p.PublicKey
	mu        sync.RWMutex
	onRecv    RecvFunc
}

func newLoopbackNetwork(localKey p2p.PublicKey) Network {
	return &loopbackNetwork{
		localAddr: p2p.NewPeerID(localKey),
		localKey:  localKey,
		onRecv:    NoOpRecvFunc,
	}
}

func (n *loopbackNetwork) Tell(ctx context.Context, dst Addr, data []byte) error {
	if dst != n.localAddr {
		return ErrNoAddrWithPrefix
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onRecv(dst, dst, data)
	return nil
}

func (n *loopbackNetwork) OnRecv(fn RecvFunc) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if fn == nil {
		fn = NoOpRecvFunc
	}
	n.onRecv = fn
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

func (n *loopbackNetwork) MTU(context.Context, Addr) int {
	return 1 << 20
}

func (n *loopbackNetwork) WaitReady(ctx context.Context) error {
	return nil
}

func (n *loopbackNetwork) LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error) {
	if addr == n.localAddr {
		return n.localKey, nil
	}
	return nil, ErrPublicKeyNotFound
}
