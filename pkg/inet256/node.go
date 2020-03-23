package inet256

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/simplemux"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
)

var (
	ErrAddrUnreachable  = errors.New("address is unreachable")
	ErrNoAddrWithPrefix = errors.New("no address found with that prefix")
)

type Params struct {
	p2p.PrivateKey
	Swarms   map[string]p2p.SecureSwarm
	Peers    PeerStore
	Networks []NetworkSpec
}

type Node struct {
	params Params

	memrealm *memswarm.Realm
	memswarm *memswarm.Swarm

	swarms   map[string]p2p.SecureSwarm
	networks []Network

	addrMap sync.Map
}

func NewNode(params Params) *Node {
	swarms := params.Swarms
	memrealm := memswarm.NewRealm()
	memsw := memrealm.NewSwarmWithKey(params.PrivateKey)
	swarms["virtual"] = memsw

	s := multiswarm.NewSecure(swarms)
	mux := simplemux.MultiplexSwarm(s)

	networks := make([]Network, len(params.Networks))
	for i, nspec := range params.Networks {
		s, err := mux.OpenSecure(nspec.Name)
		if err != nil {
			panic(err)
		}
		networks[i] = nspec.Factory(NetworkParams{
			Swarm: s.(p2p.SecureSwarm),
			Peers: params.Peers,
		})
	}

	return &Node{
		params: params,

		memswarm: memsw,
		memrealm: memrealm,

		networks: networks,
	}
}

func (n *Node) SendTo(ctx context.Context, dst Addr, data []byte) error {
	network := n.whichNetwork(ctx, dst)
	if network == nil {
		return ErrAddrUnreachable
	}
	return network.SendTo(ctx, dst, data)
}

func (n *Node) OnRecv(fn RecvFunc) {
	for _, netw := range n.networks {
		netw.OnRecv(fn)
	}
}

func (n *Node) AddrWithPrefix(ctx context.Context, prefix []byte, nbits int) (addr Addr, err error) {
	addr, _, err = n.addrWithPrefix(ctx, prefix, nbits)
	return addr, err
}

func (n *Node) LocalAddr() Addr {
	return NewAddr(n.params.PrivateKey.Public())
}

// MinMTU returns the minumum which will work for all the peers.
func (n *Node) MinMTU() int {
	mtu := 0
	for _, netw := range n.networks {
		x := netw.MinMTU()
		if x < mtu {
			mtu = x
		}
	}
	return mtu
}

func (n *Node) NewVirtual() *Node {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	mems := n.memrealm.NewSwarmWithKey(privateKey)
	swarms := map[string]p2p.SecureSwarm{
		"virtual": mems,
	}

	addrs := []string{}
	data, err := n.memswarm.LocalAddrs()[0].MarshalText()
	if err != nil {
		panic(err)
	}
	addrs = append(addrs, string(data))

	peerStore := NewPeerStore()
	peerStore.AddPeer(n.LocalAddr())
	peerStore.PutAddrs(n.LocalAddr(), addrs)

	return NewNode(Params{
		PrivateKey: privateKey,
		Swarms:     swarms,
		Networks:   n.params.Networks,
		Peers:      peerStore,
	})
}

func (n *Node) Close() error {
	errs := []error{}
	for _, network := range n.networks {
		if err := network.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

func (n *Node) addrWithPrefix(ctx context.Context, prefix []byte, nbits int) (addr Addr, network Network, err error) {
	addrs := make([]Addr, len(n.networks))
	errs := make([]error, len(n.networks))

	ctx, cf := context.WithCancel(ctx)
	defer cf()
	wg := sync.WaitGroup{}
	wg.Add(len(n.networks))
	for i, network := range n.networks[:] {
		i := i
		network := network
		go func() {
			addrs[i], errs[i] = network.AddrWithPrefix(ctx, prefix, nbits)
			if errs[i] == nil {
				cf()
			}
			wg.Done()
		}()
	}
	wg.Wait()

	for i, addr := range addrs {
		if errs[i] == nil {
			return addr, n.networks[i], nil
		}
	}
	return Addr{}, nil, fmt.Errorf("errors occurred %v", addrs)
}

func (n *Node) whichNetwork(ctx context.Context, addr Addr) Network {
	x, exists := n.addrMap.Load(addr)
	if exists {
		return x.(Network)
	}

	_, network, err := n.addrWithPrefix(ctx, addr[:], len(addr)*8)
	if err != nil {
		return nil
	}
	n.addrMap.Store(addr, network)
	return network
}
