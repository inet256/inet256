package inet256

import (
	"context"
	"fmt"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
	"github.com/sirupsen/logrus"
)

// PeerSwarm is the type of a p2p.Swarm which uses p2p.PeerIDs as addresses
type PeerSwarm = peerswarm.Swarm

// RecvFunc can be passed to Network.OnRecv as a callback to receive messages
type RecvFunc func(src, dst Addr, data []byte)

// NoOpRecvFunc does nothing
func NoOpRecvFunc(src, dst Addr, data []byte) {}

// PeerSet stores information about peers
type PeerSet interface {
	ListPeers() []p2p.PeerID
	Contains(id p2p.PeerID) bool
}

// PeerStore stores information about peers
type PeerStore interface {
	ListPeers() []p2p.PeerID
	ListAddrs(p2p.PeerID) []string
	Contains(p2p.PeerID) bool
}

// Network is a network for sending messages between peers
type Network interface {
	Tell(ctx context.Context, addr Addr, data []byte) error
	OnRecv(fn RecvFunc)
	MTU(ctx context.Context, addr Addr) int

	LookupPublicKey(ctx context.Context, addr Addr) (p2p.PublicKey, error)
	FindAddr(ctx context.Context, prefix []byte, nbits int) (Addr, error)

	Close() error
}

// NetworkParams are passed to a NetworkFactory to create a Network.
// This type really defines the problem domain quite well. Essentially
// it is a set of one-hop peers and a means to send messages to them.
type NetworkParams struct {
	Swarm PeerSwarm
	Peers PeerSet
}

// NetworkFactory is a constructor for a network
type NetworkFactory func(NetworkParams) Network

// NetworkSpec is a name associated with a network factory
type NetworkSpec struct {
	Index   int
	Name    string
	Factory NetworkFactory
}

type multiNetwork struct {
	networks []Network

	mu      sync.Mutex
	addrMap sync.Map
}

func newMultiNetwork(networks []Network) Network {
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
