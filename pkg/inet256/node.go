package inet256

import (
	"context"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/intmux"
	"github.com/brendoncarroll/go-p2p/s/aggswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
)

type Params struct {
	p2p.PrivateKey
	Swarms   map[string]p2p.SecureSwarm
	Peers    PeerStore
	Networks []NetworkSpec
}

type Node interface {
	Network
	TransportAddrs() []string
	ListOneHop() []p2p.PeerID
}

type node struct {
	params Params

	baseSwarm p2p.SecureSwarm
	network   Network
}

func NewNode(params Params) Node {
	baseSwarm := multiswarm.NewSecure(params.Swarms)
	mux := intmux.WrapSecureSwarm(baseSwarm)

	// create multi network
	networks := make([]Network, len(params.Networks))
	for i, nspec := range params.Networks {
		s := mux.Open(nspec.Index)
		s = aggswarm.NewSecure(s, (1<<16)-1)
		ps := NewPeerSwarm(s, NewAddrSource(s, params.Peers))
		networks[i] = nspec.Factory(NetworkParams{
			PrivateKey: params.PrivateKey,
			Swarm:      ps,
			Peers:      params.Peers,
		})
	}

	var network Network = newMultiNetwork(networks...)
	// apply top layer of security
	network = newSecureNetwork(params.PrivateKey, network)
	// loopback
	network = newChainNetwork(
		newLoopbackNetwork(params.PrivateKey.Public()),
		network,
	)

	return &node{
		params:    params,
		baseSwarm: baseSwarm,
		network:   network,
	}
}

func (n *node) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.network.Tell(ctx, dst, data)
}

func (n *node) OnRecv(fn RecvFunc) {
	n.network.OnRecv(fn)
}

func (n *node) FindAddr(ctx context.Context, prefix []byte, nbits int) (addr Addr, err error) {
	return n.network.FindAddr(ctx, prefix, nbits)
}

func (n *node) LocalAddr() Addr {
	return NewAddr(n.params.PrivateKey.Public())
}

func (n *node) LookupPublicKey(ctx context.Context, target Addr) (p2p.PublicKey, error) {
	return n.network.LookupPublicKey(ctx, target)
}

func (n *node) MTU(ctx context.Context, target Addr) int {
	return n.network.MTU(ctx, target)
}

func (n *node) TransportAddrs() (ret []string) {
	for _, addr := range n.baseSwarm.LocalAddrs() {
		data, _ := addr.MarshalText()
		ret = append(ret, string(data))
	}
	return ret
}

func (n *node) ListOneHop() []Addr {
	return n.params.Peers.ListPeers()
}

func (n *node) WaitReady(ctx context.Context) error {
	return n.network.WaitReady(ctx)
}

func (n *node) Close() error {
	return n.network.Close()
}
