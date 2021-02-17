package inet256

import (
	"context"
	"fmt"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/noiseswarm"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/sirupsen/logrus"
)

// PeerSwarm is the type of a p2p.Swarm which uses p2p.PeerIDs as addresses
type PeerSwarm = peerswarm.Swarm

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

	Close() error
}

// WaitInit has the WaitInit method
type WaitInit interface {
	Network
	// WaitInit block's until the Network has initialized.
	// After WaitInit has completed, all addresses should be reachable from all other addresses.
	WaitInit(ctx context.Context) error
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

func newMultiNetwork(networks []Network) Network {
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
		return ErrAddrUnreachable
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
		return nil, ErrAddrUnreachable
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
		return ErrAddrUnreachable
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
	return Addr{}, ErrAddrUnreachable
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

func (n *loopbackNetwork) LookupPublicKey(ctx context.Context, addr Addr) (PublicKey, error) {
	if addr == n.localAddr {
		return n.localKey, nil
	}
	return nil, ErrPublicKeyNotFound
}

type noise2PeerSwarm struct {
	*noiseswarm.Swarm
}

func newSecureNetwork(privateKey p2p.PrivateKey, x Network) Network {
	insecSw := SwarmFromNetwork(x, privateKey.Public())
	noiseSw := noiseswarm.New(insecSw, privateKey)
	secnet := networkFromSwarm(noise2PeerSwarm{noiseSw}, x.FindAddr)
	return secnet
}

func (s noise2PeerSwarm) OnTell(fn p2p.TellHandler) {
	s.Swarm.OnTell(func(x *p2p.Message) {
		fn(&p2p.Message{
			Src:     x.Src.(noiseswarm.Addr).ID,
			Dst:     x.Dst.(noiseswarm.Addr).ID,
			Payload: x.Payload,
		})
	})
}

func (s noise2PeerSwarm) TellPeer(ctx context.Context, id p2p.PeerID, data []byte) error {
	return s.Swarm.Tell(ctx, noiseswarm.Addr{ID: id, Addr: id}, data)
}

func (s noise2PeerSwarm) LocalAddrs() (ys []p2p.Addr) {
	for _, x := range s.Swarm.LocalAddrs() {
		ys = append(ys, x.(noiseswarm.Addr).ID)
	}
	return ys
}

func (s noise2PeerSwarm) LocalID() p2p.PeerID {
	return s.Swarm.LocalAddrs()[0].(noiseswarm.Addr).ID
}
