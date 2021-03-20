package inet256

import (
	"context"
	"time"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/p/intmux"
	"github.com/brendoncarroll/go-p2p/s/fragswarm"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
)

const TransportMTU = (1 << 16) - 1

type Params struct {
	p2p.PrivateKey
	Swarms   map[string]p2p.SecureSwarm
	Peers    PeerStore
	Networks []NetworkSpec
}

type Node interface {
	Network

	ListOneHop() []p2p.PeerID
}

type node struct {
	params Params

	transportSwarm p2p.SecureSwarm
	basePeerSwarm  *peerSwarm
	network        Network
}

func NewNode(params Params) Node {
	transportSwarm := multiswarm.NewSecure(params.Swarms)
	basePeerSwarm := newPeerSwarm(transportSwarm, params.Peers)
	mux := intmux.WrapSecureSwarm(basePeerSwarm)

	// create multi network
	networks := make([]Network, len(params.Networks))
	for i, nspec := range params.Networks {
		s := mux.Open(nspec.Index)
		s = fragswarm.NewSecure(s, TransportMTU)
		ps := castPeerSwarm(s)
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
		params:         params,
		transportSwarm: transportSwarm,
		basePeerSwarm:  basePeerSwarm,
		network:        network,
	}
}

func (n *node) Tell(ctx context.Context, dst Addr, data []byte) error {
	return n.network.Tell(ctx, dst, data)
}

func (n *node) Recv(fn RecvFunc) error {
	return n.network.Recv(fn)
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
	for _, addr := range n.transportSwarm.LocalAddrs() {
		data, _ := addr.MarshalText()
		ret = append(ret, string(data))
	}
	return ret
}

func (n *node) LastSeen(id p2p.PeerID) map[string]time.Time {
	return n.basePeerSwarm.LastSeen(id)
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
