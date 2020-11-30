package inet256

import (
	"context"

	"github.com/pkg/errors"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/simplemux"
	"github.com/brendoncarroll/go-p2p/s/memswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/peerswarm"
)

var ErrAddrUnreachable = errors.New("address is unreachable")

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
	network  Network
}

func NewNode(params Params) *Node {
	swarms := params.Swarms
	var memrealm *memswarm.Realm
	var memsw *memswarm.Swarm
	if swarms["virtual"] == nil {
		memrealm = memswarm.NewRealm()
		memsw = memrealm.NewSwarmWithKey(params.PrivateKey)
		swarms["virtual"] = memsw
	}

	baseSwarm := multiswarm.NewSecure(swarms)
	mux := simplemux.MultiplexSwarm(baseSwarm)

	networks := make([]Network, len(params.Networks))
	for i, nspec := range params.Networks {
		s, err := mux.OpenSecure(nspec.Name)
		if err != nil {
			panic(err)
		}
		ps := peerswarm.NewSwarm(s, newAddrSource(s, params.Peers))
		networks[i] = nspec.Factory(NetworkParams{
			Swarm: ps,
			Peers: params.Peers,
		})
	}

	return &Node{
		params:   params,
		memrealm: memrealm,
		memswarm: memsw,
		network:  newMultiNetwork(networks),
	}
}

func (n *Node) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.network.Tell(ctx, dst, data)
}

func (n *Node) OnRecv(fn RecvFunc) {
	n.network.OnRecv(fn)
}

func (n *Node) FindAddr(ctx context.Context, prefix []byte, nbits int) (addr Addr, err error) {
	return n.network.FindAddr(ctx, prefix, nbits)
}

func (n *Node) LocalAddr() Addr {
	return NewAddr(n.params.PrivateKey.Public())
}

func (n *Node) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return n.network.LookupPublicKey(ctx, target)
}

func (n *Node) MTU(ctx context.Context, target Addr) int {
	return n.network.MTU(ctx, target)
}

func (n *Node) NewVirtual(privateKey p2p.PrivateKey) *Node {
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
	return n.network.Close()
}
